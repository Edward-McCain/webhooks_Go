package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/auth"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/config"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/postgres"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/repository"
	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DB.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	endpoints := repository.NewEndpointRepository(pool)
	events := repository.NewEventRepository(pool)
	deliveries := repository.NewDeliveryRepository(pool)

	now := time.Now().UTC()
	secret, _ := auth.GenerateEndpointSecret()
	publicID, _ := auth.GeneratePublicID()

	ep := &domain.Endpoint{
		ID: uuid.New(), PublicID: publicID, Name: "Demo Endpoint",
		TargetURL: "http://api:8080/demo/webhook?mode=200", Secret: secret,
		Active: true, RateLimit: 100, CreatedAt: now, UpdatedAt: now,
	}
	if err := endpoints.Create(ctx, ep); err != nil {
		log.Fatal(err)
	}

	statuses := []domain.EventStatus{
		domain.EventStatusDelivered,
		domain.EventStatusFailed,
		domain.EventStatusRetrying,
		domain.EventStatusPending,
	}
	for i, st := range statuses {
		payload, _ := json.Marshal(map[string]any{"seed": true, "n": i + 1})
		ev := &domain.Event{
			ID: uuid.New(), EndpointID: ep.ID, ExternalEventID: fmt.Sprintf("seed_%d", i+1),
			EventType: "seed.demo", Payload: payload, Status: st, CreatedAt: now.Add(-time.Duration(i) * time.Hour), UpdatedAt: now,
		}
		if err := events.Create(ctx, ev); err != nil {
			log.Fatal(err)
		}
		delStatus := domain.DeliveryStatus(st)
		del := &domain.Delivery{
			ID: uuid.New(), EventID: ev.ID, EndpointID: ep.ID, Status: delStatus,
			AttemptCount: 1, CreatedAt: ev.CreatedAt, UpdatedAt: now,
		}
		if st == domain.EventStatusDelivered {
			code := 200
			del.ResponseStatus = &code
			del.DeliveredAt = &now
		}
		if err := deliveries.Create(ctx, del); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("seeded endpoint public_id=%s secret=%s\n", ep.PublicID, secret)
	_ = os.Stdout
}
