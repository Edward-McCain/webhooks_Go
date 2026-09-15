package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/auth"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/config"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	hmacsig "github.com/Edward-McCain/webhooks_Go/backend/internal/hmac"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/kafka"
	redisx "github.com/Edward-McCain/webhooks_Go/backend/internal/redis"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/repository"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/ssrf"
	"github.com/google/uuid"
)

type Services struct {
	Endpoints  *EndpointService
	Events     *EventService
	Deliveries *DeliveryService
	APIKeys    *APIKeyService
	Ingest     *IngestService
	Stats      *StatsService
}

func NewServices(
	cfg *config.Config,
	endpoints *repository.EndpointRepository,
	events *repository.EventRepository,
	deliveries *repository.DeliveryRepository,
	apiKeys *repository.APIKeyRepository,
	rdb *redisx.Client,
	producer *kafka.Producer,
	log *slog.Logger,
) *Services {
	return &Services{
		Endpoints:  NewEndpointService(cfg, endpoints, log),
		Events:     NewEventService(events, deliveries, producer, log),
		Deliveries: NewDeliveryService(deliveries, events, log),
		APIKeys:    NewAPIKeyService(apiKeys, log),
		Ingest:     NewIngestService(cfg, endpoints, events, deliveries, rdb, producer, log),
		Stats:      NewStatsService(events),
	}
}

type EndpointService struct {
	cfg  *config.Config
	repo *repository.EndpointRepository
	log  *slog.Logger
}

func NewEndpointService(cfg *config.Config, repo *repository.EndpointRepository, log *slog.Logger) *EndpointService {
	return &EndpointService{cfg: cfg, repo: repo, log: log}
}

type CreateEndpointInput struct {
	Name      string `json:"name"`
	TargetURL string `json:"target_url"`
	RateLimit int    `json:"rate_limit"`
}

type CreateEndpointResult struct {
	Endpoint *domain.Endpoint `json:"endpoint"`
	Secret   string           `json:"secret"`
}

func (s *EndpointService) Create(ctx context.Context, in CreateEndpointInput) (*CreateEndpointResult, error) {
	if in.Name == "" {
		return nil, domain.NewAppError("VALIDATION_ERROR", "name is required", 400, nil)
	}
	if err := ssrf.ValidateTargetURL(in.TargetURL); err != nil {
		return nil, domain.ErrInvalidURL
	}
	rateLimit := in.RateLimit
	if rateLimit <= 0 {
		rateLimit = s.cfg.RateLimit.Requests
	}

	publicID, err := auth.GeneratePublicID()
	if err != nil {
		return nil, err
	}
	secret, err := auth.GenerateEndpointSecret()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	ep := &domain.Endpoint{
		ID:        uuid.New(),
		PublicID:  publicID,
		Name:      in.Name,
		TargetURL: in.TargetURL,
		Secret:    secret,
		Active:    true,
		RateLimit: rateLimit,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, ep); err != nil {
		return nil, err
	}
	return &CreateEndpointResult{Endpoint: ep, Secret: secret}, nil
}

func (s *EndpointService) Get(ctx context.Context, id uuid.UUID) (*domain.Endpoint, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *EndpointService) List(ctx context.Context, limit int, cursor *time.Time) ([]domain.Endpoint, error) {
	return s.repo.List(ctx, limit, cursor)
}

type UpdateEndpointInput struct {
	Name      *string `json:"name"`
	TargetURL *string `json:"target_url"`
	Active    *bool   `json:"active"`
	RateLimit *int    `json:"rate_limit"`
}

func (s *EndpointService) Update(ctx context.Context, id uuid.UUID, in UpdateEndpointInput) (*domain.Endpoint, error) {
	ep, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		ep.Name = *in.Name
	}
	if in.TargetURL != nil {
		if err := ssrf.ValidateTargetURL(*in.TargetURL); err != nil {
			return nil, domain.ErrInvalidURL
		}
		ep.TargetURL = *in.TargetURL
	}
	if in.Active != nil {
		ep.Active = *in.Active
	}
	if in.RateLimit != nil {
		if *in.RateLimit <= 0 {
			return nil, domain.NewAppError("VALIDATION_ERROR", "rate_limit must be > 0", 400, nil)
		}
		ep.RateLimit = *in.RateLimit
	}
	ep.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, ep); err != nil {
		return nil, err
	}
	return ep, nil
}

func (s *EndpointService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

type EventService struct {
	events     *repository.EventRepository
	deliveries *repository.DeliveryRepository
	producer   *kafka.Producer
	log        *slog.Logger
}

func NewEventService(events *repository.EventRepository, deliveries *repository.DeliveryRepository, producer *kafka.Producer, log *slog.Logger) *EventService {
	return &EventService{events: events, deliveries: deliveries, producer: producer, log: log}
}

func (s *EventService) Get(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	return s.events.GetByID(ctx, id)
}

func (s *EventService) List(ctx context.Context, f repository.EventListFilter) ([]domain.Event, error) {
	return s.events.List(ctx, f)
}

func (s *EventService) Retry(ctx context.Context, id uuid.UUID) error {
	ev, err := s.events.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ev.Status != domain.EventStatusFailed && ev.Status != domain.EventStatusDelivered {
		return domain.NewAppError("INVALID_STATE", "event cannot be retried in current status", 400, nil)
	}

	del, err := s.deliveries.GetByEventID(ctx, ev.ID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	ev.Status = domain.EventStatusPending
	ev.UpdatedAt = now
	if err := s.events.UpdateStatus(ctx, ev.ID, domain.EventStatusPending); err != nil {
		return err
	}

	del.Status = domain.DeliveryStatusPending
	del.NextRetryAt = nil
	del.UpdatedAt = now
	if err := s.deliveries.Update(ctx, del); err != nil {
		return err
	}

	return s.producer.PublishEvent(ctx, domain.KafkaEventMessage{
		EventID:    ev.ID.String(),
		EndpointID: ev.EndpointID.String(),
		CreatedAt:  now.Format(time.RFC3339Nano),
	})
}

type DeliveryService struct {
	deliveries *repository.DeliveryRepository
	events     *repository.EventRepository
	log        *slog.Logger
}

func NewDeliveryService(deliveries *repository.DeliveryRepository, events *repository.EventRepository, log *slog.Logger) *DeliveryService {
	return &DeliveryService{deliveries: deliveries, events: events, log: log}
}

func (s *DeliveryService) Get(ctx context.Context, id uuid.UUID) (*domain.Delivery, error) {
	return s.deliveries.GetByID(ctx, id)
}

func (s *DeliveryService) GetByEventID(ctx context.Context, eventID uuid.UUID) (*domain.Delivery, error) {
	return s.deliveries.GetByEventID(ctx, eventID)
}

func (s *DeliveryService) List(ctx context.Context, limit int, cursor *time.Time, endpointID *uuid.UUID, status *domain.DeliveryStatus) ([]domain.Delivery, error) {
	return s.deliveries.List(ctx, limit, cursor, endpointID, status)
}

func (s *DeliveryService) ListAttempts(ctx context.Context, deliveryID uuid.UUID) ([]domain.DeliveryAttempt, error) {
	return s.deliveries.ListAttempts(ctx, deliveryID)
}

type APIKeyService struct {
	repo *repository.APIKeyRepository
	log  *slog.Logger
}

func NewAPIKeyService(repo *repository.APIKeyRepository, log *slog.Logger) *APIKeyService {
	return &APIKeyService{repo: repo, log: log}
}

type CreateAPIKeyResult struct {
	Key       *domain.APIKey `json:"key"`
	Plaintext string         `json:"plaintext"`
}

func (s *APIKeyService) Create(ctx context.Context, name string) (*CreateAPIKeyResult, error) {
	if name == "" {
		return nil, domain.NewAppError("VALIDATION_ERROR", "name is required", 400, nil)
	}
	plaintext, prefix, hash, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, err
	}
	key := &domain.APIKey{
		ID:        uuid.New(),
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   hash,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}
	return &CreateAPIKeyResult{Key: key, Plaintext: plaintext}, nil
}

func (s *APIKeyService) List(ctx context.Context) ([]domain.APIKey, error) {
	return s.repo.List(ctx)
}

func (s *APIKeyService) Revoke(ctx context.Context, id uuid.UUID) error {
	return s.repo.Revoke(ctx, id)
}

func (s *APIKeyService) Authenticate(ctx context.Context, token string) (*domain.APIKey, error) {
	hash := auth.HashAPIKey(token)
	key, err := s.repo.GetByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if key.RevokedAt != nil {
		return nil, domain.ErrUnauthorized
	}
	_ = s.repo.TouchLastUsed(ctx, key.ID)
	return key, nil
}

func (s *APIKeyService) EnsureBootstrap(ctx context.Context, plaintext string) error {
	if plaintext == "" {
		return nil
	}
	hash := auth.HashAPIKey(plaintext)
	if _, err := s.repo.GetByHash(ctx, hash); err == nil {
		return nil
	}
	prefix := plaintext
	if len(prefix) > 11 {
		prefix = plaintext[:11]
	}
	key := &domain.APIKey{
		ID:        uuid.New(),
		Name:      "bootstrap",
		KeyPrefix: prefix,
		KeyHash:   hash,
		CreatedAt: time.Now().UTC(),
	}
	return s.repo.Create(ctx, key)
}

type StatsService struct {
	events *repository.EventRepository
}

func NewStatsService(events *repository.EventRepository) *StatsService {
	return &StatsService{events: events}
}

func (s *StatsService) Get(ctx context.Context) (*domain.Stats, error) {
	return s.events.Stats(ctx)
}

type IngestService struct {
	cfg        *config.Config
	endpoints  *repository.EndpointRepository
	events     *repository.EventRepository
	deliveries *repository.DeliveryRepository
	rdb        *redisx.Client
	producer   *kafka.Producer
	log        *slog.Logger
}

func NewIngestService(
	cfg *config.Config,
	endpoints *repository.EndpointRepository,
	events *repository.EventRepository,
	deliveries *repository.DeliveryRepository,
	rdb *redisx.Client,
	producer *kafka.Producer,
	log *slog.Logger,
) *IngestService {
	return &IngestService{
		cfg: cfg, endpoints: endpoints, events: events, deliveries: deliveries,
		rdb: rdb, producer: producer, log: log,
	}
}

type IngestResult struct {
	Event     *domain.Event `json:"event"`
	Duplicate bool          `json:"duplicate"`
}

func (s *IngestService) Ingest(ctx context.Context, publicID string, payload []byte, eventID, eventType, signature, timestamp string) (*IngestResult, error) {
	if int64(len(payload)) > s.cfg.Delivery.MaxPayloadSize {
		return nil, domain.ErrPayloadTooLarge
	}
	if !json.Valid(payload) {
		return nil, domain.ErrInvalidPayload
	}

	ep, err := s.endpoints.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if !ep.Active {
		return nil, domain.ErrEndpointInactive
	}

	allowed, err := s.rdb.Allow(ctx, fmt.Sprintf("rl:endpoint:%s", ep.ID.String()), ep.RateLimit, s.cfg.RateLimit.Window)
	if err != nil {
		s.log.Warn("rate limit check failed, allowing request", "error", err)
	} else if !allowed {
		return nil, domain.ErrRateLimited
	}

	if err := hmacsig.Verify(ep.Secret, signature, timestamp, payload, time.Now().UTC(), hmacsig.DefaultTolerance); err != nil {
		s.log.Info("webhook signature rejected", "endpoint_id", ep.ID.String(), "reason", err.Error())
		msg := err.Error()
		if msg == "timestamp outside tolerance window" || msg == "missing timestamp" ||
			len(msg) >= 18 && msg[:18] == "invalid timestamp:" {
			return nil, domain.ErrInvalidTimestamp
		}
		return nil, domain.ErrInvalidSignature
	}

	if eventID == "" {
		eventID, err = auth.GenerateEventID()
		if err != nil {
			return nil, err
		}
	}
	if eventType == "" {
		eventType = "webhook"
	}

	now := time.Now().UTC()
	ev := &domain.Event{
		ID:              uuid.New(),
		EndpointID:      ep.ID,
		ExternalEventID: eventID,
		EventType:       eventType,
		Payload:         payload,
		Status:          domain.EventStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.events.Create(ctx, ev); err != nil {
		if err == domain.ErrConflict {
			existing, getErr := s.events.GetByExternalID(ctx, ep.ID, eventID)
			if getErr != nil {
				return nil, getErr
			}
			return &IngestResult{Event: existing, Duplicate: true}, nil
		}
		return nil, err
	}

	del := &domain.Delivery{
		ID:           uuid.New(),
		EventID:      ev.ID,
		EndpointID:   ep.ID,
		Status:       domain.DeliveryStatusPending,
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.deliveries.Create(ctx, del); err != nil {
		return nil, err
	}

	if err := s.producer.PublishEvent(ctx, domain.KafkaEventMessage{
		EventID:    ev.ID.String(),
		EndpointID: ep.ID.String(),
		CreatedAt:  now.Format(time.RFC3339Nano),
	}); err != nil {
		return nil, err
	}

	return &IngestResult{Event: ev, Duplicate: false}, nil
}
