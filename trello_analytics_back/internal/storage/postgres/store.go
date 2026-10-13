package postgres

import "github.com/jackc/pgx/v5/pgxpool"

// Store provides PostgreSQL persistence helpers for import/sync.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pgx pool. pool must be non-nil.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Pool exposes the underlying pool (tests / advanced use).
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}
