package db

import (
	"fmt"
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

type PgStorage struct {
	db *pgxpool.Pool
}

func NewPgStorage(db_url string) (*PgStorage, error) {
	db, err := pgxpool.New(context.Background(), db_url)
	if err != nil {
        return nil, fmt.Errorf("failed to create pgx pool: %w", err)
    }
	return &PgStorage{db}, err
}

func (pgs *PgStorage) Close () {
	pgs.db.Close();
}
