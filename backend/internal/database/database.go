package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %w", err)
	}

	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return db, nil
}