package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/api"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/config"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/kafka"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/logger"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/middleware"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/postgres"
	redisx "github.com/Edward-McCain/webhooks_Go/backend/internal/redis"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/repository"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/service"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/ssrf"
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

	if err := postgres.Migrate(cfg.DB.URL); err != nil {
		log.Error("migrate database", "error", err)
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DB.URL)
	if err != nil {
		log.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := redisx.New(cfg.Redis.URL)
	if err != nil {
		log.Error("connect redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic, log)
	defer producer.Close()

	endpoints := repository.NewEndpointRepository(pool)
	events := repository.NewEventRepository(pool)
	deliveries := repository.NewDeliveryRepository(pool)
	apiKeys := repository.NewAPIKeyRepository(pool)

	svc := service.NewServices(cfg, endpoints, events, deliveries, apiKeys, rdb, producer, log)
	if err := svc.APIKeys.EnsureBootstrap(ctx, cfg.BootstrapAPIKey); err != nil {
		log.Error("bootstrap api key", "error", err)
		os.Exit(1)
	}

	handler := api.NewHandler(svc, cfg.Delivery.MaxPayloadSize, func(r *http.Request) error {
		if err := pool.Ping(r.Context()); err != nil {
			return err
		}
		return rdb.Ping(r.Context())
	})

	var h http.Handler = handler.Routes(middleware.APIKeyAuth(svc.APIKeys))
	h = middleware.MaxBytes(cfg.Delivery.MaxPayloadSize + 1024)(h)
	h = middleware.Metrics(h)
	h = middleware.Logging(log)(h)
	h = middleware.RequestID(h)
	h = middleware.Recover(log)(h)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      h,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		log.Info("api listening", "port", cfg.HTTP.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down api")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("api shutdown error", "error", err)
	}
	log.Info("api stopped", "uptime_hint", time.Now().UTC().Format(time.RFC3339))
}
