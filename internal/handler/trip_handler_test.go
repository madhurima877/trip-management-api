package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"trip-management-api/internal/model"
	"trip-management-api/internal/service"
)

type stubTripService struct {
	createErr   error
	listTrips   []model.Trip
	listTotal   int
	listFilter  model.TripFilter
	createCalls int
}

func (s *stubTripService) Create(context.Context, model.CreateTrip) (model.Trip, error) {
	s.createCalls++
	return model.Trip{}, s.createErr
}

func (s *stubTripService) List(_ context.Context, filter model.TripFilter) ([]model.Trip, int, error) {
	s.listFilter = filter
	return s.listTrips, s.listTotal, nil
}

func (*stubTripService) Get(context.Context, string) (model.Trip, error) {
	return model.Trip{}, errors.New("not implemented")
}

func (*stubTripService) UpdateStatus(context.Context, string, model.Status) (model.Trip, error) {
	return model.Trip{}, errors.New("not implemented")
}

func (*stubTripService) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

func newTestHandler(service TripService) http.Handler {
	mux := http.NewServeMux()
	NewTripHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux)
	return mux
}

func TestCreateRejectsUnknownFields(t *testing.T) {
	stub := &stubTripService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/trips", strings.NewReader(`{"trip_number":"T-1","unexpected":true}`))
	newTestHandler(stub).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if stub.createCalls != 0 {
		t.Fatal("service called for invalid request")
	}
	if !strings.Contains(recorder.Body.String(), `"error":{"code":"invalid_request"`) {
		t.Fatalf("unexpected error response: %s", recorder.Body.String())
	}
}

func TestListPassesPaginationAndStatusFilter(t *testing.T) {
	stub := &stubTripService{
		listTrips: []model.Trip{{ID: "trip-id", TripNumber: "T-1", Status: model.StatusAssigned, CreatedAt: time.Now()}},
		listTotal: 12,
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/trips?limit=5&offset=10&status=ASSIGNED", nil)
	newTestHandler(stub).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if stub.listFilter.Limit != 5 || stub.listFilter.Offset != 10 || stub.listFilter.Status != model.StatusAssigned {
		t.Fatalf("filter = %+v, want limit=5 offset=10 status=ASSIGNED", stub.listFilter)
	}
	if !strings.Contains(recorder.Body.String(), `"total":12`) {
		t.Fatalf("pagination total missing: %s", recorder.Body.String())
	}
}

func TestDuplicateCreateReturnsConflict(t *testing.T) {
	stub := &stubTripService{createErr: service.ErrDuplicateTrip}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/trips", strings.NewReader(`{"trip_number":"T-1","source":"A","destination":"B","driver_name":"Driver"}`))
	newTestHandler(stub).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"trip_number_conflict"`) {
		t.Fatalf("unexpected error response: %s", recorder.Body.String())
	}
}
