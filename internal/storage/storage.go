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

var (
	ErrConflict        = errors.New("conflict")
	ErrPaymentRequired = errors.New("PaymentRequired")
)

type DB struct {
	db   *sql.DB
	auth map[string]bool
}

type bonuses struct {
	login string
	sum   float32
	out   float32
}

type spending struct {
	user_login   string
	order_number string
	date         time.Time
	sum          float32
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

func (db *DB) GetOrders(ctx context.Context, login string) ([]model.OrderSchema, error) {

	rows, err := db.db.QueryContext(ctx, querySelectOrders, login)
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

func (db *DB) SetBonuses(ctx context.Context, accrualList []model.AccrualSchema) error {

	orders, err := findLogins(db, accrualList)
	if err != nil {
		return err
	}

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	for _, data := range orders {
		err = insertBonuses(ctx, tx, bonuses{
			login: data.user,
			sum:   data.Accrual,
			out:   0,
		})
		if err != nil {
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

func insertBonuses(ctx context.Context, tx *sql.Tx, bonuses bonuses) error {

	_, err := tx.ExecContext(ctx, queryInsertBonuses,
		bonuses.login,
		bonuses.sum,
		bonuses.out)

	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (db *DB) GetBalance(ctx context.Context, login string) (model.BalaneSchema, error) {
	row := db.db.QueryRowContext(context.Background(), queryBalance, login)

	var balance model.BalaneSchema
	err := row.Scan(&balance.Current, &balance.WithDrawn)
	if err != nil {
		return balance, err
	}

	return balance, nil
}

func (db *DB) WithdrawBonuses(ctx context.Context, login string, data model.WithdrawSchema) error {

	balance, err := db.GetBalance(ctx, login)
	if err != nil {
		return err
	}

	if balance.Current < data.Sum {
		return ErrPaymentRequired
	}

	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	err = insertBonuses(ctx, tx, bonuses{
		login: login,
		sum:   -data.Sum,
		out:   data.Sum,
	})
	if err != nil {
		return err
	}

	err = insertSpending(ctx, tx, spending{
		user_login:   login,
		order_number: data.Order,
		date:         time.Now(),
		sum:          data.Sum,
	})
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func insertSpending(ctx context.Context, tx *sql.Tx, data spending) error {

	_, err := tx.ExecContext(ctx, queryInsertSpending,
		data.user_login,
		data.order_number,
		data.date,
		data.sum)

	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (db *DB) Getwithdrawels(ctx context.Context, login string) ([]model.WithdrawalsSchema, error) {

	rows, err := db.db.QueryContext(ctx, querySelectSpending, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.WithdrawalsSchema, 0)
	for rows.Next() {
		var order model.WithdrawalsSchema
		err := rows.Scan(&order.Order,
			&order.Sum,
			&order.Date)
		if err != nil {
			return nil, err
		}
		order.ProcessedAt = order.Date.Format(time.RFC3339)
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return orders, nil
}
