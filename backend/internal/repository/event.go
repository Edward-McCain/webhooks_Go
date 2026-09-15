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

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

type EventListFilter struct {
	Status     *domain.EventStatus
	EndpointID *uuid.UUID
	EventType  *string
	From       *time.Time
	To         *time.Time
	Cursor     *time.Time
	Limit      int
}

func (r *EventRepository) Create(ctx context.Context, ev *domain.Event) error {
	const q = `
		INSERT INTO events (id, endpoint_id, external_event_id, event_type, payload, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, q,
		ev.ID, ev.EndpointID, ev.ExternalEventID, ev.EventType, ev.Payload, ev.Status, ev.CreatedAt, ev.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return domain.ErrConflict
	}
	return err
}

func (r *EventRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	const q = `
		SELECT id, endpoint_id, external_event_id, event_type, payload, status, created_at, updated_at
		FROM events WHERE id = $1`
	var ev domain.Event
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&ev.ID, &ev.EndpointID, &ev.ExternalEventID, &ev.EventType, &ev.Payload, &ev.Status, &ev.CreatedAt, &ev.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *EventRepository) GetByExternalID(ctx context.Context, endpointID uuid.UUID, externalID string) (*domain.Event, error) {
	const q = `
		SELECT id, endpoint_id, external_event_id, event_type, payload, status, created_at, updated_at
		FROM events WHERE endpoint_id = $1 AND external_event_id = $2`
	var ev domain.Event
	err := r.pool.QueryRow(ctx, q, endpointID, externalID).Scan(
		&ev.ID, &ev.EndpointID, &ev.ExternalEventID, &ev.EventType, &ev.Payload, &ev.Status, &ev.CreatedAt, &ev.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *EventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.EventStatus) error {
	const q = `UPDATE events SET status = $2, updated_at = $3 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, status, time.Now().UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEventNotFound
	}
	return nil
}

func (r *EventRepository) List(ctx context.Context, f EventListFilter) ([]domain.Event, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	const q = `
		SELECT id, endpoint_id, external_event_id, event_type, payload, status, created_at, updated_at
		FROM events
		WHERE ($1::text IS NULL OR status = $1)
		  AND ($2::uuid IS NULL OR endpoint_id = $2)
		  AND ($3::text IS NULL OR event_type = $3)
		  AND ($4::timestamptz IS NULL OR created_at >= $4)
		  AND ($5::timestamptz IS NULL OR created_at <= $5)
		  AND ($6::timestamptz IS NULL OR created_at < $6)
		ORDER BY created_at DESC, id DESC
		LIMIT $7`

	var status *string
	if f.Status != nil {
		s := string(*f.Status)
		status = &s
	}

	rows, err := r.pool.Query(ctx, q, status, f.EndpointID, f.EventType, f.From, f.To, f.Cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Event, 0)
	for rows.Next() {
		var ev domain.Event
		if err := rows.Scan(&ev.ID, &ev.EndpointID, &ev.ExternalEventID, &ev.EventType, &ev.Payload, &ev.Status, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (r *EventRepository) Stats(ctx context.Context) (*domain.Stats, error) {
	const q = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'DELIVERED') AS delivered,
			COUNT(*) FILTER (WHERE status = 'FAILED') AS failed,
			COUNT(*) FILTER (WHERE status = 'RETRYING') AS retrying,
			COUNT(*) FILTER (WHERE status = 'PENDING') AS pending,
			COUNT(*) FILTER (WHERE status = 'PROCESSING') AS processing
		FROM events`

	var s domain.Stats
	if err := r.pool.QueryRow(ctx, q).Scan(
		&s.TotalEvents, &s.Delivered, &s.Failed, &s.Retrying, &s.Pending, &s.Processing,
	); err != nil {
		return nil, err
	}

	completed := s.Delivered + s.Failed
	if completed > 0 {
		s.SuccessRate = float64(s.Delivered) / float64(completed) * 100
	}

	const avgQ = `
		SELECT COALESCE(AVG(duration_ms), 0)
		FROM delivery_attempts
		WHERE status = 'SUCCESS'`
	if err := r.pool.QueryRow(ctx, avgQ).Scan(&s.AvgDeliveryTimeMS); err != nil {
		return nil, err
	}
	return &s, nil
}
