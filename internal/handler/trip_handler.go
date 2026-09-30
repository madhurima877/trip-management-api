package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"trip-management-api/internal/model"
	"trip-management-api/internal/service"
)

type TripService interface {
	Create(context.Context, model.CreateTrip) (model.Trip, error)
	List(context.Context, model.TripFilter) ([]model.Trip, int, error)
	Get(context.Context, string) (model.Trip, error)
	UpdateStatus(context.Context, string, model.Status) (model.Trip, error)
	Delete(context.Context, string) error
}

type TripHandler struct {
	service TripService
	logger  *slog.Logger
}

func NewTripHandler(tripService TripService, logger *slog.Logger) *TripHandler {
	return &TripHandler{service: tripService, logger: logger}
}

func (h *TripHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /trips", h.create)
	mux.HandleFunc("GET /trips", h.list)
	mux.HandleFunc("GET /trips/{id}", h.get)
	mux.HandleFunc("PATCH /trips/{id}/status", h.updateStatus)
	mux.HandleFunc("DELETE /trips/{id}", h.delete)
}

func (h *TripHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateTrip
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	trip, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Location", "/trips/"+trip.ID)
	writeJSON(w, http.StatusCreated, trip)
}

func (h *TripHandler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	for key := range query {
		if key != "status" && key != "limit" && key != "offset" {
			writeError(w, http.StatusBadRequest, "invalid_request", "unsupported query parameter: "+key)
			return
		}
	}
	filter := model.TripFilter{Limit: 20}
	if status := query.Get("status"); status != "" {
		filter.Status = model.Status(status)
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, http.StatusBadRequest, "invalid_request", "limit must be between 1 and 100")
			return
		}
		filter.Limit = limit
	}
	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "offset must be a non-negative integer")
			return
		}
		filter.Offset = offset
	}
	trips, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":       trips,
		"pagination": map[string]int{"limit": filter.Limit, "offset": filter.Offset, "total": total},
	})
}

func (h *TripHandler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "invalid_request", "id must be a UUID")
		return
	}
	trip, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func (h *TripHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "invalid_request", "id must be a UUID")
		return
	}
	var input struct {
		Status model.Status `json:"status"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	trip, err := h.service.UpdateStatus(r.Context(), id, input.Status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func (h *TripHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		writeError(w, http.StatusBadRequest, "invalid_request", "id must be a UUID")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TripHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, service.ErrDuplicateTrip):
		writeError(w, http.StatusConflict, "trip_number_conflict", err.Error())
	case errors.Is(err, service.ErrTripNotFound):
		writeError(w, http.StatusNotFound, "trip_not_found", err.Error())
	case errors.Is(err, service.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "invalid_status_transition", err.Error())
	default:
		h.logger.Error("trip request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validID(id string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(id))
}
