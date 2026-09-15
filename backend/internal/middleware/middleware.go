package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/auth"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/logger"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/metrics"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/service"
)

type ctxKey int

const (
	RequestIDKey ctxKey = iota
	APIKeyKey
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := GetRequestID(r.Context())
			reqLog := logger.WithRequestID(log, reqID)
			ctx := logger.IntoContext(r.Context(), reqLog)
			ww := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r.WithContext(ctx))
			reqLog.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		metrics.ObserveHTTP(r.Method, r.URL.Path, ww.status, time.Since(start))
	})
}

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", "panic", rec, "path", r.URL.Path)
					http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func MaxBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func APIKeyAuth(apiKeys *service.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := auth.BearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeUnauthorized(w, GetRequestID(r.Context()))
				return
			}
			key, err := apiKeys.Authenticate(r.Context(), token)
			if err != nil {
				writeUnauthorized(w, GetRequestID(r.Context()))
				return
			}
			ctx := context.WithValue(r.Context(), APIKeyKey, key)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"Unauthorized","request_id":"` + requestID + `"}}`))
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
