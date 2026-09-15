package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/config"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/delivery"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/kafka"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/metrics"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/repository"
	"github.com/google/uuid"
)

type Pool struct {
	cfg        *config.Config
	consumer   *kafka.Consumer
	endpoints  *repository.EndpointRepository
	events     *repository.EventRepository
	deliveries *repository.DeliveryRepository
	client     *delivery.Client
	jobs       chan domain.KafkaEventMessage
	log        *slog.Logger
	wg         sync.WaitGroup
}

func NewPool(
	cfg *config.Config,
	consumer *kafka.Consumer,
	endpoints *repository.EndpointRepository,
	events *repository.EventRepository,
	deliveries *repository.DeliveryRepository,
	client *delivery.Client,
	log *slog.Logger,
) *Pool {
	return &Pool{
		cfg:        cfg,
		consumer:   consumer,
		endpoints:  endpoints,
		events:     events,
		deliveries: deliveries,
		client:     client,
		jobs:       make(chan domain.KafkaEventMessage, cfg.Worker.Count*2),
		log:        log,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.cfg.Worker.Count; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	p.wg.Add(1)
	go p.consume(ctx)
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) consume(ctx context.Context) {
	defer p.wg.Done()
	for {
		msg, raw, err := p.consumer.Fetch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				close(p.jobs)
				return
			}
			p.log.Error("kafka fetch failed", "error", err)
			metrics.KafkaMessagesFailedTotal.Inc()
			time.Sleep(time.Second)
			continue
		}

		metrics.QueueDepth.Set(float64(len(p.jobs)))
		select {
		case <-ctx.Done():
			_ = p.consumer.Commit(context.Background(), raw)
			close(p.jobs)
			return
		case p.jobs <- msg:
			if err := p.consumer.Commit(ctx, raw); err != nil {
				p.log.Error("kafka commit failed", "error", err)
			}
		}
	}
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	defer func() {
		if rec := recover(); rec != nil {
			p.log.Error("worker panic recovered", "worker_id", id, "panic", rec)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			metrics.QueueDepth.Set(float64(len(p.jobs)))
			metrics.ActiveWorkers.Inc()
			p.process(ctx, job)
			metrics.ActiveWorkers.Dec()
			metrics.KafkaMessagesProcessedTotal.Inc()
		}
	}
}

func (p *Pool) process(ctx context.Context, job domain.KafkaEventMessage) {
	eventID, err := uuid.Parse(job.EventID)
	if err != nil {
		p.log.Error("invalid event id in kafka message", "event_id", job.EventID)
		metrics.KafkaMessagesFailedTotal.Inc()
		return
	}

	ev, err := p.events.GetByID(ctx, eventID)
	if err != nil {
		p.log.Error("load event failed", "event_id", job.EventID, "error", err)
		return
	}

	// Skip terminal states (idempotent reprocessing).
	if ev.Status == domain.EventStatusDelivered || ev.Status == domain.EventStatusFailed {
		return
	}

	ep, err := p.endpoints.GetByID(ctx, ev.EndpointID)
	if err != nil {
		p.log.Error("load endpoint failed", "endpoint_id", ev.EndpointID.String(), "error", err)
		return
	}

	del, err := p.deliveries.GetByEventID(ctx, ev.ID)
	if err != nil {
		p.log.Error("load delivery failed", "event_id", ev.ID.String(), "error", err)
		return
	}

	if del.Status == domain.DeliveryStatusRetrying && del.NextRetryAt != nil && del.NextRetryAt.After(time.Now().UTC()) {
		// Not ready yet — republish later would be ideal; for now sleep briefly is avoided.
		// Re-queue by publishing is handled by retry scheduler path inside markRetry.
		return
	}

	_ = p.events.UpdateStatus(ctx, ev.ID, domain.EventStatusProcessing)
	del.Status = domain.DeliveryStatusProcessing
	del.UpdatedAt = time.Now().UTC()
	_ = p.deliveries.Update(ctx, del)

	attemptNo := del.AttemptCount + 1
	result := p.client.Deliver(ctx, ep.TargetURL, ep.Secret, ev.ExternalEventID, ev.EventType, ev.Payload)
	metrics.WebhookDeliveryAttemptsTotal.Inc()
	metrics.WebhookDeliveryDuration.Observe(result.Duration.Seconds())

	body := result.Body
	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}
	var statusPtr *int
	if result.StatusCode > 0 {
		statusPtr = &result.StatusCode
	}
	var errStr *string
	if result.Err != nil {
		s := result.Err.Error()
		errStr = &s
	}

	attemptStatus := domain.AttemptStatusFailed
	if result.Err == nil && result.StatusCode >= 200 && result.StatusCode < 300 {
		attemptStatus = domain.AttemptStatusSuccess
	} else if result.Err != nil && result.StatusCode == 0 {
		attemptStatus = domain.AttemptStatusTimeout
	}

	attempt := &domain.DeliveryAttempt{
		ID:             uuid.New(),
		DeliveryID:     del.ID,
		AttemptNumber:  attemptNo,
		Status:         attemptStatus,
		ResponseStatus: statusPtr,
		ResponseBody:   bodyPtr,
		DurationMS:     result.Duration.Milliseconds(),
		Error:          errStr,
		CreatedAt:      time.Now().UTC(),
	}
	if err := p.deliveries.CreateAttempt(ctx, attempt); err != nil {
		p.log.Error("persist attempt failed", "error", err)
	}

	del.AttemptCount = attemptNo
	del.ResponseStatus = statusPtr
	del.ResponseBody = bodyPtr
	del.UpdatedAt = time.Now().UTC()

	if attemptStatus == domain.AttemptStatusSuccess {
		now := time.Now().UTC()
		del.Status = domain.DeliveryStatusDelivered
		del.DeliveredAt = &now
		del.NextRetryAt = nil
		_ = p.deliveries.Update(ctx, del)
		_ = p.events.UpdateStatus(ctx, ev.ID, domain.EventStatusDelivered)
		metrics.WebhooksDeliveredTotal.Inc()
		p.log.Info("webhook delivered",
			"event_id", ev.ID.String(),
			"endpoint_id", ep.ID.String(),
			"attempt", attemptNo,
			"duration_ms", result.Duration.Milliseconds(),
		)
		return
	}

	retryable := result.Retryable || (result.Err != nil && delivery.IsRetryableError(result.Err))
	if result.StatusCode > 0 {
		retryable = delivery.IsRetryableStatus(result.StatusCode)
	}

	if !retryable || attemptNo > p.cfg.Delivery.MaxRetries {
		del.Status = domain.DeliveryStatusFailed
		del.NextRetryAt = nil
		_ = p.deliveries.Update(ctx, del)
		_ = p.events.UpdateStatus(ctx, ev.ID, domain.EventStatusFailed)
		metrics.WebhooksFailedTotal.Inc()
		p.log.Info("webhook failed permanently",
			"event_id", ev.ID.String(),
			"endpoint_id", ep.ID.String(),
			"attempt", attemptNo,
		)
		return
	}

	delay := delivery.Backoff(attemptNo, delivery.DefaultBackoffSchedule, 0.2)
	next := time.Now().UTC().Add(delay)
	del.Status = domain.DeliveryStatusRetrying
	del.NextRetryAt = &next
	_ = p.deliveries.Update(ctx, del)
	_ = p.events.UpdateStatus(ctx, ev.ID, domain.EventStatusRetrying)
	metrics.WebhookRetriesTotal.Inc()

	p.log.Info("webhook retry scheduled",
		"event_id", ev.ID.String(),
		"endpoint_id", ep.ID.String(),
		"attempt", attemptNo,
		"next_retry_at", next.Format(time.RFC3339),
	)

	// Schedule republish after backoff.
	go p.scheduleRetry(ev.ID, ep.ID, delay)
}

func (p *Pool) scheduleRetry(eventID, endpointID uuid.UUID, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	<-timer.C
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.republish(ctx, eventID, endpointID); err != nil {
		p.log.Error("failed to republish retry", "event_id", eventID.String(), "error", err)
	}
}

func (p *Pool) republish(ctx context.Context, eventID, endpointID uuid.UUID) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = nil
		}
	}()
	select {
	case p.jobs <- domain.KafkaEventMessage{
		EventID:    eventID.String(),
		EndpointID: endpointID.String(),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
