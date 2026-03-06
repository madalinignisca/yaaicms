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

// VariantStore handles database operations for responsive image variants.
type VariantStore struct {
	db *sql.DB
}

// NewVariantStore creates a new VariantStore with the given database connection.
func NewVariantStore(db *sql.DB) *VariantStore {
	return &VariantStore{db: db}
}

const variantColumns = `id, media_id, name, width, height, s3_key, content_type, size_bytes, created_at`

func scanVariant(scanner interface{ Scan(...any) error }) (*models.MediaVariant, error) {
	var v models.MediaVariant
	err := scanner.Scan(
		&v.ID, &v.MediaID, &v.Name, &v.Width, &v.Height,
		&v.S3Key, &v.ContentType, &v.SizeBytes, &v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateBatch inserts multiple variants in a single transaction. Used after
// generating all responsive sizes for an uploaded image.
func (s *VariantStore) CreateBatch(variants []models.MediaVariant) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin variant batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO media_variants (media_id, name, width, height, s3_key, content_type, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return fmt.Errorf("prepare variant insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, v := range variants {
		if _, err := stmt.Exec(v.MediaID, v.Name, v.Width, v.Height, v.S3Key, v.ContentType, v.SizeBytes); err != nil {
			return fmt.Errorf("insert variant %s: %w", v.Name, err)
		}
	}

	return tx.Commit()
}

// FindByMediaID returns all variants for a given media item, ordered by width.
func (s *VariantStore) FindByMediaID(mediaID uuid.UUID) ([]models.MediaVariant, error) {
	rows, err := s.db.Query(`
		SELECT `+variantColumns+`
		FROM media_variants
		WHERE media_id = $1
		ORDER BY width ASC
	`, mediaID)
	if err != nil {
		return nil, fmt.Errorf("find variants by media: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var variants []models.MediaVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		variants = append(variants, *v)
	}
	return variants, rows.Err()
}

// FindByMediaIDs returns variants for multiple media items at once, keyed by
// media ID. Used for batch resolution in post listings.
func (s *VariantStore) FindByMediaIDs(mediaIDs []uuid.UUID) (map[uuid.UUID][]models.MediaVariant, error) {
	if len(mediaIDs) == 0 {
		return nil, nil
	}

	// Build placeholder list for IN clause.
	placeholders := ""
	args := make([]any, len(mediaIDs))
	for i, id := range mediaIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := s.db.Query(`
		SELECT `+variantColumns+`
		FROM media_variants
		WHERE media_id IN (`+placeholders+`)
		ORDER BY media_id, width ASC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("find variants by media ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[uuid.UUID][]models.MediaVariant)
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		result[v.MediaID] = append(result[v.MediaID], *v)
	}
	return result, rows.Err()
}

// DeleteByMediaID removes all variants for a media item. Returns the deleted
// variants so the caller can clean up S3 objects.
func (s *VariantStore) DeleteByMediaID(mediaID uuid.UUID) ([]models.MediaVariant, error) {
	rows, err := s.db.Query(`
		DELETE FROM media_variants
		WHERE media_id = $1
		RETURNING `+variantColumns, mediaID)
	if err != nil {
		return nil, fmt.Errorf("delete variants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var variants []models.MediaVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan deleted variant: %w", err)
		}
		variants = append(variants, *v)
	}
	return variants, rows.Err()
}

// ListMediaWithoutVariants returns media IDs for images that have no variants
// yet. Used by the bulk regeneration endpoint.
func (s *VariantStore) ListMediaWithoutVariants(limit int) ([]uuid.UUID, error) {
	rows, err := s.db.Query(`
		SELECT m.id FROM media m
		LEFT JOIN media_variants mv ON mv.media_id = m.id
		WHERE m.content_type LIKE 'image/%'
		  AND m.content_type NOT IN ('image/svg+xml', 'image/gif')
		  AND mv.id IS NULL
		ORDER BY m.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list media without variants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan media id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
