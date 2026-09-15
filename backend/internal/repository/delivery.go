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

type DeliveryRepository struct {
	pool *pgxpool.Pool
}

func NewDeliveryRepository(pool *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{pool: pool}
}

func (r *DeliveryRepository) Create(ctx context.Context, d *domain.Delivery) error {
	const q = `
		INSERT INTO deliveries (
			id, event_id, endpoint_id, status, attempt_count, response_status, response_body,
			next_retry_at, delivered_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := r.pool.Exec(ctx, q,
		d.ID, d.EventID, d.EndpointID, d.Status, d.AttemptCount, d.ResponseStatus, d.ResponseBody,
		d.NextRetryAt, d.DeliveredAt, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *DeliveryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Delivery, error) {
	const q = `
		SELECT id, event_id, endpoint_id, status, attempt_count, response_status, response_body,
		       next_retry_at, delivered_at, created_at, updated_at
		FROM deliveries WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *DeliveryRepository) GetByEventID(ctx context.Context, eventID uuid.UUID) (*domain.Delivery, error) {
	const q = `
		SELECT id, event_id, endpoint_id, status, attempt_count, response_status, response_body,
		       next_retry_at, delivered_at, created_at, updated_at
		FROM deliveries WHERE event_id = $1`
	return r.scanOne(ctx, q, eventID)
}

func (r *DeliveryRepository) Update(ctx context.Context, d *domain.Delivery) error {
	const q = `
		UPDATE deliveries SET
			status = $2,
			attempt_count = $3,
			response_status = $4,
			response_body = $5,
			next_retry_at = $6,
			delivered_at = $7,
			updated_at = $8
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q,
		d.ID, d.Status, d.AttemptCount, d.ResponseStatus, d.ResponseBody,
		d.NextRetryAt, d.DeliveredAt, d.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDeliveryNotFound
	}
	return nil
}

func (r *DeliveryRepository) List(ctx context.Context, limit int, cursor *time.Time, endpointID *uuid.UUID, status *domain.DeliveryStatus) ([]domain.Delivery, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var statusStr *string
	if status != nil {
		s := string(*status)
		statusStr = &s
	}

	const q = `
		SELECT id, event_id, endpoint_id, status, attempt_count, response_status, response_body,
		       next_retry_at, delivered_at, created_at, updated_at
		FROM deliveries
		WHERE ($1::uuid IS NULL OR endpoint_id = $1)
		  AND ($2::text IS NULL OR status = $2)
		  AND ($3::timestamptz IS NULL OR created_at < $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4`

	rows, err := r.pool.Query(ctx, q, endpointID, statusStr, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Delivery, 0)
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *DeliveryRepository) CreateAttempt(ctx context.Context, a *domain.DeliveryAttempt) error {
	const q = `
		INSERT INTO delivery_attempts (
			id, delivery_id, attempt_number, status, response_status, response_body, duration_ms, error, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.pool.Exec(ctx, q,
		a.ID, a.DeliveryID, a.AttemptNumber, a.Status, a.ResponseStatus, a.ResponseBody, a.DurationMS, a.Error, a.CreatedAt,
	)
	return err
}

func (r *DeliveryRepository) ListAttempts(ctx context.Context, deliveryID uuid.UUID) ([]domain.DeliveryAttempt, error) {
	const q = `
		SELECT id, delivery_id, attempt_number, status, response_status, response_body, duration_ms, error, created_at
		FROM delivery_attempts
		WHERE delivery_id = $1
		ORDER BY attempt_number ASC`

	rows, err := r.pool.Query(ctx, q, deliveryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.DeliveryAttempt, 0)
	for rows.Next() {
		var a domain.DeliveryAttempt
		if err := rows.Scan(
			&a.ID, &a.DeliveryID, &a.AttemptNumber, &a.Status, &a.ResponseStatus, &a.ResponseBody, &a.DurationMS, &a.Error, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *DeliveryRepository) scanOne(ctx context.Context, q string, arg any) (*domain.Delivery, error) {
	row := r.pool.QueryRow(ctx, q, arg)
	d, err := scanDelivery(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrDeliveryNotFound
	}
	return d, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDelivery(row scannable) (*domain.Delivery, error) {
	var d domain.Delivery
	err := row.Scan(
		&d.ID, &d.EventID, &d.EndpointID, &d.Status, &d.AttemptCount, &d.ResponseStatus, &d.ResponseBody,
		&d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
