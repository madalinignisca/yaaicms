// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"yaaicms/internal/models"
)

// revisionColumns lists all columns for content_revisions SELECTs.
const revisionColumns = `id, content_id, title, slug, body, body_format, excerpt,
	status, meta_description, meta_keywords, featured_image_id, category_id,
	revision_title, revision_log, created_by, created_at`

// RevisionStore provides access to content revision data in PostgreSQL.
type RevisionStore struct {
	db *sql.DB
}

// NewRevisionStore creates a new RevisionStore backed by the given database.
func NewRevisionStore(db *sql.DB) *RevisionStore {
	return &RevisionStore{db: db}
}

// scanRevision scans a single content_revisions row into a ContentRevision.
func scanRevision(scanner interface{ Scan(...any) error }) (*models.ContentRevision, error) {
	var r models.ContentRevision
	err := scanner.Scan(
		&r.ID, &r.ContentID, &r.Title, &r.Slug, &r.Body, &r.BodyFormat,
		&r.Excerpt, &r.Status, &r.MetaDescription, &r.MetaKeywords,
		&r.FeaturedImageID, &r.CategoryID, &r.RevisionTitle, &r.RevisionLog, &r.CreatedBy, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Create inserts a new content revision and returns it with the generated ID.
func (s *RevisionStore) Create(rev *models.ContentRevision) (*models.ContentRevision, error) {
	row := s.db.QueryRow(`
		INSERT INTO content_revisions (
			content_id, title, slug, body, body_format, excerpt, status,
			meta_description, meta_keywords, featured_image_id, category_id,
			revision_title, revision_log, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING `+revisionColumns,
		rev.ContentID, rev.Title, rev.Slug, rev.Body, rev.BodyFormat, rev.Excerpt,
		rev.Status, rev.MetaDescription, rev.MetaKeywords, rev.FeaturedImageID,
		rev.CategoryID, rev.RevisionTitle, rev.RevisionLog, rev.CreatedBy,
	)
	return scanRevision(row)
}

// ListByContentID returns all revisions for a content item, newest first.
func (s *RevisionStore) ListByContentID(contentID uuid.UUID) ([]*models.ContentRevision, error) {
	rows, err := s.db.Query(`
		SELECT `+revisionColumns+`
		FROM content_revisions
		WHERE content_id = $1
		ORDER BY created_at DESC
	`, contentID)
	if err != nil {
		return nil, fmt.Errorf("list revisions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var revisions []*models.ContentRevision
	for rows.Next() {
		r, err := scanRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan revision: %w", err)
		}
		revisions = append(revisions, r)
	}
	return revisions, rows.Err()
}

// FindByID returns a single revision by its ID.
func (s *RevisionStore) FindByID(id uuid.UUID) (*models.ContentRevision, error) {
	row := s.db.QueryRow(`
		SELECT `+revisionColumns+`
		FROM content_revisions
		WHERE id = $1
	`, id)
	return scanRevision(row)
}

// UpdateMeta updates the revision title and changelog for a given revision.
func (s *RevisionStore) UpdateMeta(id uuid.UUID, title, changelog string) error {
	_, err := s.db.Exec(`
		UPDATE content_revisions
		SET revision_title = $1, revision_log = $2
		WHERE id = $3
	`, title, changelog, id)
	if err != nil {
		return fmt.Errorf("update revision meta: %w", err)
	}
	return nil
}

// Count returns the number of revisions for a content item.
func (s *RevisionStore) Count(contentID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM content_revisions WHERE content_id = $1
	`, contentID).Scan(&count)
	return count, err
}
