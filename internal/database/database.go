// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

// Package database handles PostgreSQL connection management and migration
// execution using goose. It provides a Connect function that returns a
// ready-to-use *sql.DB pool and a Migrate function for schema management.
package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations
var embedMigrations embed.FS

// Connect opens a PostgreSQL connection pool using the provided DSN.
// It verifies the connection with a ping before returning.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("database open: %w", err)
	}

	// Set connection pool limits to prevent resource exhaustion.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify the connection is alive.
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database ping: %w", err)
	}

	slog.Info("database connected")
	return db, nil
}

// Migrate runs all pending goose migrations from the embedded SQL files.
// Migrations are embedded at compile time so no external files are needed
// at runtime.
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	slog.Info("database migrations applied")
	return nil
}
