package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EndpointRepository struct {
	pool *pgxpool.Pool
}

func NewEndpointRepository(pool *pgxpool.Pool) *EndpointRepository {
	return &EndpointRepository{pool: pool}
}

func (r *EndpointRepository) Create(ctx context.Context, ep *domain.Endpoint) error {
	const q = `
		INSERT INTO endpoints (id, public_id, name, target_url, secret, active, rate_limit, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, q,
		ep.ID, ep.PublicID, ep.Name, ep.TargetURL, ep.Secret, ep.Active, ep.RateLimit, ep.CreatedAt, ep.UpdatedAt,
	)
	return err
}

func (r *EndpointRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error) {
	const q = `
		SELECT id, public_id, name, target_url, secret, active, rate_limit, created_at, updated_at
		FROM endpoints WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *EndpointRepository) GetByPublicID(ctx context.Context, publicID string) (*domain.Endpoint, error) {
	const q = `
		SELECT id, public_id, name, target_url, secret, active, rate_limit, created_at, updated_at
		FROM endpoints WHERE public_id = $1`
	return r.scanOne(ctx, q, publicID)
}

func (r *EndpointRepository) List(ctx context.Context, limit int, cursor *time.Time) ([]domain.Endpoint, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var (
		rows pgx.Rows
		err  error
	)
	if cursor == nil {
		const q = `
			SELECT id, public_id, name, target_url, secret, active, rate_limit, created_at, updated_at
			FROM endpoints
			ORDER BY created_at DESC, id DESC
			LIMIT $1`
		rows, err = r.pool.Query(ctx, q, limit)
	} else {
		const q = `
			SELECT id, public_id, name, target_url, secret, active, rate_limit, created_at, updated_at
			FROM endpoints
			WHERE created_at < $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`
		rows, err = r.pool.Query(ctx, q, *cursor, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Endpoint, 0)
	for rows.Next() {
		var ep domain.Endpoint
		if err := rows.Scan(&ep.ID, &ep.PublicID, &ep.Name, &ep.TargetURL, &ep.Secret, &ep.Active, &ep.RateLimit, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ep)
	}
	return out, rows.Err()
}

func (r *EndpointRepository) Update(ctx context.Context, ep *domain.Endpoint) error {
	const q = `
		UPDATE endpoints
		SET name = $2, target_url = $3, active = $4, rate_limit = $5, updated_at = $6
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, ep.ID, ep.Name, ep.TargetURL, ep.Active, ep.RateLimit, ep.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEndpointNotFound
	}
	return nil
}

func (r *EndpointRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM endpoints WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEndpointNotFound
	}
	return nil
}

func (r *EndpointRepository) scanOne(ctx context.Context, q string, arg any) (*domain.Endpoint, error) {
	var ep domain.Endpoint
	err := r.pool.QueryRow(ctx, q, arg).Scan(
		&ep.ID, &ep.PublicID, &ep.Name, &ep.TargetURL, &ep.Secret, &ep.Active, &ep.RateLimit, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrEndpointNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ep, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
