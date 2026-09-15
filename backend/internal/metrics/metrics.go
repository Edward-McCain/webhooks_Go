package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	WebhooksReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_received_total",
		Help: "Total webhooks received",
	})

	WebhooksDeliveredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_delivered_total",
		Help: "Total webhooks successfully delivered",
	})

	WebhooksFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhooks_failed_total",
		Help: "Total webhooks permanently failed",
	})

	WebhookDeliveryAttemptsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhook_delivery_attempts_total",
		Help: "Total webhook delivery attempts",
	})

	WebhookDeliveryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "webhook_delivery_duration_seconds",
		Help:    "Webhook delivery attempt latency",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})

	WebhookRetriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhook_retries_total",
		Help: "Total webhook retries scheduled",
	})

	KafkaMessagesProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_messages_processed_total",
		Help: "Total Kafka messages processed",
	})

	KafkaMessagesFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kafka_messages_failed_total",
		Help: "Total Kafka messages that failed processing",
	})

	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_workers",
		Help: "Number of currently active workers",
	})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "queue_depth",
		Help: "Current worker job queue depth",
	})
)

func ObserveHTTP(method, path string, status int, d time.Duration) {
	statusStr := strconv.Itoa(status)
	HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(d.Seconds())
}
