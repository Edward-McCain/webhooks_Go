package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	const q = `
		INSERT INTO api_keys (id, name, key_prefix, key_hash, revoked_at, last_used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, q,
		key.ID, key.Name, key.KeyPrefix, key.KeyHash, key.RevokedAt, key.LastUsedAt, key.CreatedAt,
	)
	return err
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	const q = `
		SELECT id, name, key_prefix, key_hash, revoked_at, last_used_at, created_at
		FROM api_keys WHERE key_hash = $1`
	var key domain.APIKey
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&key.ID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.RevokedAt, &key.LastUsedAt, &key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) List(ctx context.Context) ([]domain.APIKey, error) {
	const q = `
		SELECT id, name, key_prefix, key_hash, revoked_at, last_used_at, created_at
		FROM api_keys
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.APIKey, 0)
	for rows.Next() {
		var key domain.APIKey
		if err := rows.Scan(&key.ID, &key.Name, &key.KeyPrefix, &key.KeyHash, &key.RevokedAt, &key.LastUsedAt, &key.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx, `UPDATE api_keys SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *APIKeyRepository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = $2 WHERE id = $1`, id, time.Now().UTC())
	return err
}
