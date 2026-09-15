package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/domain"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/metrics"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/middleware"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/repository"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/service"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	svc            *service.Services
	maxPayloadSize int64
	readyCheck     func(r *http.Request) error
}

func NewHandler(svc *service.Services, maxPayloadSize int64, readyCheck func(*http.Request) error) *Handler {
	return &Handler{svc: svc, maxPayloadSize: maxPayloadSize, readyCheck: readyCheck}
}

func (h *Handler) Routes(authMw func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /ready", h.Ready)
	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("POST /webhooks/{public_id}", h.IngestWebhook)
	mux.HandleFunc("POST /demo/webhook", h.DemoWebhook)

	api := http.NewServeMux()
	api.HandleFunc("POST /api/v1/endpoints", h.CreateEndpoint)
	api.HandleFunc("GET /api/v1/endpoints", h.ListEndpoints)
	api.HandleFunc("GET /api/v1/endpoints/{id}", h.GetEndpoint)
	api.HandleFunc("PATCH /api/v1/endpoints/{id}", h.UpdateEndpoint)
	api.HandleFunc("DELETE /api/v1/endpoints/{id}", h.DeleteEndpoint)

	api.HandleFunc("GET /api/v1/events", h.ListEvents)
	api.HandleFunc("GET /api/v1/events/{id}", h.GetEvent)
	api.HandleFunc("POST /api/v1/events/{id}/retry", h.RetryEvent)

	api.HandleFunc("GET /api/v1/deliveries", h.ListDeliveries)
	api.HandleFunc("GET /api/v1/deliveries/{id}", h.GetDelivery)

	api.HandleFunc("GET /api/v1/stats", h.GetStats)

	api.HandleFunc("POST /api/v1/api-keys", h.CreateAPIKey)
	api.HandleFunc("GET /api/v1/api-keys", h.ListAPIKeys)
	api.HandleFunc("DELETE /api/v1/api-keys/{id}", h.RevokeAPIKey)

	mux.Handle("/api/", authMw(api))
	return mux
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.readyCheck != nil {
		if err := h.readyCheck(r); err != nil {
			writeError(w, r, domain.NewAppError("NOT_READY", "service not ready", 503, err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) IngestWebhook(w http.ResponseWriter, r *http.Request) {
	publicID := r.PathValue("public_id")
	body, err := io.ReadAll(io.LimitReader(r.Body, h.maxPayloadSize+1))
	if err != nil {
		writeError(w, r, domain.ErrInvalidPayload)
		return
	}
	if int64(len(body)) > h.maxPayloadSize {
		writeError(w, r, domain.ErrPayloadTooLarge)
		return
	}

	metrics.WebhooksReceivedTotal.Inc()
	result, err := h.svc.Ingest.Ingest(
		r.Context(),
		publicID,
		body,
		r.Header.Get("X-Event-ID"),
		r.Header.Get("X-Event-Type"),
		r.Header.Get("X-Webhook-Signature"),
		r.Header.Get("X-Webhook-Timestamp"),
	)
	if err != nil {
		writeError(w, r, err)
		return
	}

	status := http.StatusAccepted
	if result.Duplicate {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]any{
		"event_id":          result.Event.ID,
		"external_event_id": result.Event.ExternalEventID,
		"status":            result.Event.Status,
		"duplicate":         result.Duplicate,
	})
}

func (h *Handler) DemoWebhook(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	switch mode {
	case "400":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
	case "500":
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	case "timeout":
		time.Sleep(30 * time.Second)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func (h *Handler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var in service.CreateEndpointInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, domain.ErrInvalidPayload)
		return
	}
	res, err := h.svc.Endpoints.Create(r.Context(), in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"endpoint": res.Endpoint,
		"secret":   res.Secret,
	})
}

func (h *Handler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var cursor *time.Time
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339Nano, c)
		if err == nil {
			cursor = &t
		}
	}
	items, err := h.svc.Endpoints.List(r.Context(), limit, cursor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	ep, err := h.svc.Endpoints.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *Handler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	var in service.UpdateEndpointInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, domain.ErrInvalidPayload)
		return
	}
	ep, err := h.svc.Endpoints.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *Handler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	if err := h.svc.Endpoints.Delete(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := repository.EventListFilter{Limit: limit}
	if s := q.Get("status"); s != "" {
		st := domain.EventStatus(s)
		f.Status = &st
	}
	if e := q.Get("endpoint_id"); e != "" {
		id, err := uuid.Parse(e)
		if err == nil {
			f.EndpointID = &id
		}
	}
	if t := q.Get("event_type"); t != "" {
		f.EventType = &t
	}
	if c := q.Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339Nano, c)
		if err == nil {
			f.Cursor = &t
		}
	}
	items, err := h.svc.Events.List(r.Context(), f)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	ev, err := h.svc.Events.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := map[string]any{"event": ev}
	if delivery, err := h.svc.Deliveries.GetByEventID(r.Context(), ev.ID); err == nil {
		attempts, _ := h.svc.Deliveries.ListAttempts(r.Context(), delivery.ID)
		resp["delivery"] = delivery
		resp["attempts"] = attempts
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) RetryEvent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	if err := h.svc.Events.Retry(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	var cursor *time.Time
	if c := q.Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339Nano, c)
		if err == nil {
			cursor = &t
		}
	}
	var endpointID *uuid.UUID
	if e := q.Get("endpoint_id"); e != "" {
		id, err := uuid.Parse(e)
		if err == nil {
			endpointID = &id
		}
	}
	var status *domain.DeliveryStatus
	if s := q.Get("status"); s != "" {
		st := domain.DeliveryStatus(s)
		status = &st
	}
	items, err := h.svc.Deliveries.List(r.Context(), limit, cursor, endpointID, status)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	d, err := h.svc.Deliveries.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	attempts, err := h.svc.Deliveries.ListAttempts(r.Context(), d.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"delivery": d, "attempts": attempts})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Stats.Get(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, domain.ErrInvalidPayload)
		return
	}
	res, err := h.svc.APIKeys.Create(r.Context(), body.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"api_key": res.Key,
		"secret":  res.Plaintext,
		"warning": "Save this key now. It will not be shown again.",
	})
}

func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.APIKeys.List(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, r, domain.ErrValidation)
		return
	}
	if err := h.svc.APIKeys.Revoke(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.HTTPStatus, map[string]any{
			"error": map[string]any{
				"code":       appErr.Code,
				"message":    appErr.Message,
				"request_id": middleware.GetRequestID(r.Context()),
			},
		})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]any{
			"code":       "INTERNAL_ERROR",
			"message":    "Internal server error",
			"request_id": middleware.GetRequestID(r.Context()),
		},
	})
}
