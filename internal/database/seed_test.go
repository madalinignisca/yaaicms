// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package database

import (
	"testing"
)

func TestSeedIdempotent(t *testing.T) {
	db, err := Connect(testDSN())
	if err != nil {
		t.Skipf("skipping: DB not available: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Seed should be callable safely — it creates data only when tables are
	// empty. We call it twice to verify idempotency. We don't clear the
	// database first because other test packages may be running
	// concurrently against the same database.
	if err := Seed(db); err != nil {
		t.Fatalf("first Seed: %v", err)
	}
	if err := Seed(db); err != nil {
		t.Fatalf("second Seed: %v", err)
	}

	// Verify admin user exists.
	var userCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'admin@yaaicms.local'").Scan(&userCount); err != nil {
		t.Fatalf("count admin users: %v", err)
	}
	if userCount < 1 {
		t.Errorf("expected at least 1 admin user, got %d", userCount)
	}

	// Verify templates exist.
	var tmplCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM templates").Scan(&tmplCount); err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if tmplCount < 1 {
		t.Errorf("expected at least 1 template, got %d", tmplCount)
	}

	// Verify content exists.
	var contentCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM content").Scan(&contentCount); err != nil {
		t.Fatalf("count content: %v", err)
	}
	if contentCount < 1 {
		t.Errorf("expected at least 1 content item, got %d", contentCount)
	}
}
