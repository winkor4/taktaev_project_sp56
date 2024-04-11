package storage

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	db *sql.DB
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

	return &DB{db: db}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	if err := db.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}
