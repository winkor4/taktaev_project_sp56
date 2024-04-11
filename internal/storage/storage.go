package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/winkor4/taktaev_project_sp56/internal/model"
)

var ErrConflict = errors.New("conflict")

type DB struct {
	db   *sql.DB
	auth map[string]bool
}

func New(dsn string) (*DB, error) {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	out := new(DB)
	out.db = db

	ctx := context.Background()
	err = out.Ping(ctx)
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

	auth := make(map[string]bool)

	return &DB{db: db, auth: auth}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	if err := db.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}

func (db *DB) Register(login string, pass string) (bool, error) {

	tx, err := db.db.Begin()
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	result, err := tx.ExecContext(ctx, queryRegister,
		login,
		pass)
	if err != nil {
		tx.Rollback()
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return false, err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return false, err
	}

	return rowsAffected == 0, err
}

func (db *DB) Truncate() error {

	querys := make([]string, 2)
	querys[0] = "DELETE FROM users"
	querys[1] = "DELETE FROM orders"

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

func (db *DB) GetPass(login string) (string, error) {

	row := db.db.QueryRowContext(context.Background(), queryPassword, login)

	pass := new(string)
	err := row.Scan(pass)
	if err != nil {
		return "", err
	}

	return *pass, nil
}

func (db *DB) Authorisation(login string) {
	db.auth[login] = true
}

func (db *DB) Authorized(login string) bool {
	out, ok := db.auth[login]
	if !ok {
		return false
	}
	return out
}

func (db *DB) CheckOrder(login string, number string) error {

	row := db.db.QueryRowContext(context.Background(), querySelectOrder, number)

	var user string
	err := row.Scan(&user)
	if err != nil {
		return err
	}

	switch {
	case user == login:
		return nil
	default:
		return ErrConflict
	}
}

func (db *DB) UploadOrder(login string, number string) error {

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = tx.ExecContext(ctx, queryInsertdOrder,
		login,
		number,
		time.Now(),
		"NEW",
		0)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return err
}

func (db *DB) GetOrders(login string) ([]model.OrderSchema, error) {
	rows, err := db.db.QueryContext(context.Background(), querySelectOrders, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.OrderSchema, 0)
	for rows.Next() {
		var order model.OrderSchema
		err := rows.Scan(&order.Number, &order.Date, &order.Status, &order.Accrual)
		if err != nil {
			return nil, err
		}
		order.UploadedAt = order.Date.Format(time.RFC3339)
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return orders, nil
}
