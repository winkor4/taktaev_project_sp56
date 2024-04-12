package mock

import (
	"context"
	"database/sql"
	"math/rand"
	"strings"

	"github.com/winkor4/taktaev_project_sp56/internal/model"
)

type mockDB struct {
	db *sql.DB
}

func newDB(dsn string) (*mockDB, error) {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	out := new(mockDB)
	out.db = db

	ctx := context.Background()
	err = out.ping(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, migration := range migrations {
		if _, err := tx.Exec(migration); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, err
	}

	return &mockDB{db: db}, nil
}

func (db *mockDB) ping(ctx context.Context) error {
	if err := db.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (db *mockDB) newOrders() error {

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	rows, err := tx.QueryContext(ctx, queryNewOrders)
	if err != nil {
		return err
	}
	defer rows.Close()

	orders := make([]string, 0)
	for rows.Next() {
		var order string
		err := rows.Scan(&order)
		if err != nil {
			return err
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	for _, number := range orders {
		var status string
		status = "PROCESSED"
		if strings.Contains(number, "00") {
			status = "INVALID"
		}
		_, err = tx.ExecContext(ctx, queryRegister, number, status, rand.Intn(1000))
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (db *mockDB) truncate() error {

	querys := make([]string, 2)
	querys[0] = "DELETE FROM mock_orders"

	tx, err := db.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	for _, query := range querys {
		if _, err := tx.Exec(query); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (db *mockDB) getAccrual(order string) (model.AccrualSchema, error) {

	var out model.AccrualSchema

	row := db.db.QueryRowContext(context.Background(), queryGetOrder, order)

	err := row.Scan(&out.Order, &out.Status, &out.Accrual)
	if err != nil {
		return out, err
	}

	return out, nil
}
