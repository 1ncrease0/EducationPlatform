package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Storage struct {
	Pool *pgxpool.Pool
}

func NewPostgresPool(username, password, host, port, dbName string) (*Storage, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", username, password, host, port, dbName)
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return &Storage{Pool: pool}, nil
}

func (p *Storage) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

func UnwrapPgError(err error) *pgconn.PgError {
	for err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			return pgErr
		}
		err = errors.Unwrap(err)
	}
	return nil
}
