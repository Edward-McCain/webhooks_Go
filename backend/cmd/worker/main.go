package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/config"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/delivery"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/kafka"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/logger"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/postgres"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/repository"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/ssrf"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	ssrf.SetAllowPrivateTargets(cfg.AllowPrivateTargets)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DB.URL)
	if err != nil {
		log.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, log)
	defer consumer.Close()

	endpoints := repository.NewEndpointRepository(pool)
	events := repository.NewEventRepository(pool)
	deliveries := repository.NewDeliveryRepository(pool)
	client := delivery.NewClient(cfg.Delivery.HTTPTimeout, cfg.Delivery.MaxResponseBodySize)

	poolWorkers := worker.NewPool(cfg, consumer, endpoints, events, deliveries, client, log)
	poolWorkers.Start(ctx)

	log.Info("worker started", "workers", cfg.Worker.Count, "topic", cfg.Kafka.Topic)
	<-ctx.Done()
	log.Info("shutting down worker")

	done := make(chan struct{})
	go func() {
		poolWorkers.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("worker stopped cleanly")
	case <-time.After(cfg.ShutdownTimeout):
		log.Warn("worker shutdown timed out")
	}
}
