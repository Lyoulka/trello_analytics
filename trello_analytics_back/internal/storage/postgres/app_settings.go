package postgres

import (
	"context"
	"fmt"
	"time"
)

// AppSetting is a non-secret key/value row.
type AppSetting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
	UpdatedBy *string
}

// ListAppSettings returns all app_settings rows.
func (s *Store) ListAppSettings(ctx context.Context) ([]AppSetting, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT key, value, updated_at, updated_by::text
		FROM app_settings
		ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list app settings: %w", err)
	}
	defer rows.Close()
	var out []AppSetting
	for rows.Next() {
		var a AppSetting
		if err := rows.Scan(&a.Key, &a.Value, &a.UpdatedAt, &a.UpdatedBy); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpsertAppSetting sets a non-secret setting key.
func (s *Store) UpsertAppSetting(ctx context.Context, key, value string, updatedBy *string) error {
	if key == "" {
		return fmt.Errorf("upsert app setting: empty key")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at, updated_by)
		VALUES ($1, $2, now(), $3::uuid)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = now(),
			updated_by = EXCLUDED.updated_by
	`, key, value, updatedBy)
	if err != nil {
		return fmt.Errorf("upsert app setting %s: %w", key, err)
	}
	return nil
}
