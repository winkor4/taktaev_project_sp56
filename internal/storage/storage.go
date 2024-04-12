package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

	querys := make([]string, 4)
	querys[0] = "DELETE FROM users"
	querys[1] = "DELETE FROM orders"
	querys[2] = "DELETE FROM bonuses"
	querys[3] = "DELETE FROM spending"

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
	_, err = tx.ExecContext(ctx, queryInsertOrder,
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

func (db *DB) OrdersToRefresh() ([]string, error) {
	rows, err := db.db.QueryContext(context.Background(), queryOrdersToRefresh)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]string, 0)
	for rows.Next() {
		var order string
		err := rows.Scan(&order)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return orders, nil
}

func (db *DB) UpdateOrders(accrualList []model.AccrualSchema) error {

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	ctx := context.Background()
	for _, data := range accrualList {
		_, err = tx.ExecContext(ctx, queryUpdateOrder,
			data.Status,
			data.Accrual,
			data.Order)

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

func (db *DB) SetBonuses(accrualList []model.AccrualSchema) error {

	orders, err := findLogins(db, accrualList)
	if err != nil {
		return err
	}

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	ctx := context.Background()
	for _, data := range orders {
		_, err = tx.ExecContext(ctx, queryInsertBonuses,
			data.user,
			data.Accrual,
			0)

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

type accrualLogin struct {
	user string
	model.AccrualSchema
}

func findLogins(db *DB, accrualList []model.AccrualSchema) ([]accrualLogin, error) {

	var param string
	for i, data := range accrualList {
		param = param + fmt.Sprintf("'%s'", data.Order)
		if i < len(accrualList)-1 {
			param = param + ", "
		}
	}
	query := strings.ReplaceAll(queryLogins, "$1", param)

	rows, err := db.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]accrualLogin, 0)
	for rows.Next() {
		var order accrualLogin
		err := rows.Scan(&order.user, &order.Order)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	for i, order := range orders {
		for _, data := range accrualList {
			if data.Order == order.Order {
				orders[i].Accrual = data.Accrual
				orders[i].Status = data.Status
				break
			}
		}
	}

	return orders, nil
}

func (db *DB) GetBalance(login string) (model.BalaneSchema, error) {
	row := db.db.QueryRowContext(context.Background(), queryBalance, login)

	var balance model.BalaneSchema
	err := row.Scan(&balance.Current, &balance.WithDrawn)
	if err != nil {
		return balance, err
	}

	return balance, nil
}
