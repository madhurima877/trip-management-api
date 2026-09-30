package model

import (
	"errors"
	"time"
)

type Status string

const (
	StatusCreated   Status = "CREATED"
	StatusAssigned  Status = "ASSIGNED"
	StatusInTransit Status = "IN_TRANSIT"
	StatusCompleted Status = "COMPLETED"
	StatusCancelled Status = "CANCELLED"
)

var ValidStatuses = map[Status]struct{}{
	StatusCreated:   {},
	StatusAssigned:  {},
	StatusInTransit: {},
	StatusCompleted: {},
	StatusCancelled: {},
}

func CanTransition(from, to Status) bool {
	switch from {
	case StatusCreated:
		return to == StatusAssigned || to == StatusCancelled
	case StatusAssigned:
		return to == StatusInTransit || to == StatusCancelled
	case StatusInTransit:
		return to == StatusCompleted || to == StatusCancelled
	default:
		return false
	}
}

var (
	ErrDuplicateTrip     = errors.New("trip number already exists")
	ErrTripNotFound      = errors.New("trip not found")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type Trip struct {
	ID          string    `json:"id"`
	TripNumber  string    `json:"trip_number"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	DriverName  string    `json:"driver_name"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTrip struct {
	TripNumber  string `json:"trip_number"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	DriverName  string `json:"driver_name"`
}

type TripFilter struct {
	Status Status
	Limit  int
	Offset int
}
