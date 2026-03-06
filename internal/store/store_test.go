// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

// store_test.go provides a shared test database helper for all store
// integration tests. Tests are skipped if PostgreSQL is not available.
package store

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"yaaicms/internal/database"
)

// testDSN returns the PostgreSQL connection string for testing.
// Uses environment variables with defaults matching docker-compose.yml.
func testDSN() string {
	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "yaaicms")
	pass := envOr("POSTGRES_PASSWORD", "changeme")
	name := envOr("POSTGRES_DB", "yaaicms")
	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=disable"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// testDB opens a connection to the test database and runs migrations.
// If the database is unavailable, the test is skipped. A cleanup
// function is registered to close the connection when the test finishes.
func testDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := testDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot open DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("skipping integration test: DB not reachable: %v", err)
	}

	// Run migrations to ensure the schema is current.
	if err := database.Migrate(db); err != nil {
		_ = db.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Downgrade goose global state.
	goose.SetBaseFS(nil)

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// cleanUsers removes test users by email pattern. Call in t.Cleanup().
func cleanUsers(t *testing.T, db *sql.DB, emails ...string) {
	t.Helper()
	for _, email := range emails {
		_, _ = db.Exec("DELETE FROM users WHERE email = $1", email)
	}
}

// cleanContent removes test content by slug pattern. Call in t.Cleanup().
func cleanContent(t *testing.T, db *sql.DB, slugs ...string) {
	t.Helper()
	for _, slug := range slugs {
		_, _ = db.Exec("DELETE FROM content WHERE slug = $1", slug)
	}
}

// cleanTemplates removes test templates by name pattern. Call in t.Cleanup().
func cleanTemplates(t *testing.T, db *sql.DB, names ...string) {
	t.Helper()
	for _, name := range names {
		_, _ = db.Exec("DELETE FROM templates WHERE name = $1", name)
	}
}

// cleanMedia removes all test media for a given uploader. Call in t.Cleanup().
func cleanMediaByKey(t *testing.T, db *sql.DB, s3keys ...string) {
	t.Helper()
	for _, key := range s3keys {
		_, _ = db.Exec("DELETE FROM media WHERE s3_key = $1", key)
	}
}
