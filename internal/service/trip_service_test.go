package service

import (
	"testing"

	"trip-management-api/internal/model"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from model.Status
		to   model.Status
		want bool
	}{
		{model.StatusCreated, model.StatusAssigned, true},
		{model.StatusAssigned, model.StatusInTransit, true},
		{model.StatusInTransit, model.StatusCompleted, true},
		{model.StatusCreated, model.StatusCancelled, true},
		{model.StatusInTransit, model.StatusCancelled, true},
		{model.StatusCreated, model.StatusInTransit, false},
		{model.StatusAssigned, model.StatusCreated, false},
		{model.StatusCompleted, model.StatusCancelled, false},
		{model.StatusCancelled, model.StatusAssigned, false},
	}
	for _, test := range tests {
		if got := model.CanTransition(test.from, test.to); got != test.want {
			t.Errorf("CanTransition(%q, %q) = %t, want %t", test.from, test.to, got, test.want)
		}
	}
}
