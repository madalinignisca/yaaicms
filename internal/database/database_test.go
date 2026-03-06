// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

// Package database tests cover PostgreSQL connection and migration execution.
// These are integration tests that require a running PostgreSQL instance.
package database

import (
	"os"
	"testing"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func testDSN() string {
	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "yaaicms")
	pass := envOr("POSTGRES_PASSWORD", "changeme")
	name := envOr("POSTGRES_DB", "yaaicms")
	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=disable"
}

func TestConnect(t *testing.T) {
	db, err := Connect(testDSN())
	if err != nil {
		t.Skipf("skipping: DB not available: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify connection pool settings.
	if db.Stats().MaxOpenConnections != 25 {
		t.Errorf("max open conns: got %d, want 25", db.Stats().MaxOpenConnections)
	}

	// Verify connection is alive.
	if err := db.Ping(); err != nil {
		t.Errorf("ping failed after Connect: %v", err)
	}
}

func TestConnectInvalidDSN(t *testing.T) {
	_, err := Connect("postgres://invalid:invalid@localhost:1/nonexistent?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Error("expected error for invalid DSN")
	}
}

func TestMigrate(t *testing.T) {
	db, err := Connect(testDSN())
	if err != nil {
		t.Skipf("skipping: DB not available: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Migrate should be idempotent — running twice shouldn't error.
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify key tables exist.
	tables := []string{"users", "content", "templates", "media", "cache_invalidation_log"}
	for _, table := range tables {
		var exists bool
		err := db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table,
		).Scan(&exists)
		if err != nil {
			t.Errorf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist after migration", table)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := Connect(testDSN())
	if err != nil {
		t.Skipf("skipping: DB not available: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Run migrations twice — should not error.
	if err := Migrate(db); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}
