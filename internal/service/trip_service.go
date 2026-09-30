package service

import (
	"context"
	"errors"
	"strings"

	"trip-management-api/internal/model"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrDuplicateTrip     = model.ErrDuplicateTrip
	ErrTripNotFound      = model.ErrTripNotFound
	ErrInvalidTransition = model.ErrInvalidTransition
)

type TripRepository interface {
	Create(context.Context, model.CreateTrip) (model.Trip, error)
	List(context.Context, model.TripFilter) ([]model.Trip, int, error)
	Get(context.Context, string) (model.Trip, error)
	UpdateStatus(context.Context, string, model.Status) (model.Trip, error)
	Delete(context.Context, string) error
}

type TripService struct {
	repository TripRepository
}

func NewTripService(repository TripRepository) *TripService {
	return &TripService{repository: repository}
}

func (s *TripService) Create(ctx context.Context, input model.CreateTrip) (model.Trip, error) {
	input.TripNumber = strings.TrimSpace(input.TripNumber)
	input.Source = strings.TrimSpace(input.Source)
	input.Destination = strings.TrimSpace(input.Destination)
	input.DriverName = strings.TrimSpace(input.DriverName)
	if input.TripNumber == "" || input.Source == "" || input.Destination == "" || input.DriverName == "" ||
		len(input.TripNumber) > 100 || len(input.Source) > 255 || len(input.Destination) > 255 || len(input.DriverName) > 255 {
		return model.Trip{}, ErrInvalidInput
	}
	return s.repository.Create(ctx, input)
}

func (s *TripService) List(ctx context.Context, filter model.TripFilter) ([]model.Trip, int, error) {
	if filter.Status != "" {
		if _, ok := model.ValidStatuses[filter.Status]; !ok {
			return nil, 0, ErrInvalidInput
		}
	}
	return s.repository.List(ctx, filter)
}

func (s *TripService) Get(ctx context.Context, id string) (model.Trip, error) {
	return s.repository.Get(ctx, id)
}

func (s *TripService) UpdateStatus(ctx context.Context, id string, status model.Status) (model.Trip, error) {
	if _, ok := model.ValidStatuses[status]; !ok {
		return model.Trip{}, ErrInvalidInput
	}
	return s.repository.UpdateStatus(ctx, id, status)
}

func (s *TripService) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}
