package repository

import (
	"context"
	"database/sql"
	"errors"

	"trip-management-api/internal/model"
)

type PostgresTripRepository struct {
	db *sql.DB
}

func NewPostgresTripRepository(db *sql.DB) *PostgresTripRepository {
	return &PostgresTripRepository{db: db}
}

const tripColumns = `id, trip_number, source, destination, driver_name, status, created_at, updated_at`

func (r *PostgresTripRepository) Create(ctx context.Context, input model.CreateTrip) (model.Trip, error) {
	query := `INSERT INTO trips (trip_number, source, destination, driver_name)
		VALUES ($1, $2, $3, $4) RETURNING ` + tripColumns
	trip, err := scanTrip(r.db.QueryRowContext(ctx, query, input.TripNumber, input.Source, input.Destination, input.DriverName))
	if err != nil && isUniqueViolation(err) {
		return model.Trip{}, model.ErrDuplicateTrip
	}
	return trip, err
}

func (r *PostgresTripRepository) List(ctx context.Context, filter model.TripFilter) ([]model.Trip, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM trips WHERE ($1 = '' OR status = $1)`, string(filter.Status)).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + tripColumns + ` FROM trips
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, string(filter.Status), filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	trips := make([]model.Trip, 0)
	for rows.Next() {
		trip, err := scanTrip(rows)
		if err != nil {
			return nil, 0, err
		}
		trips = append(trips, trip)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return trips, total, nil
}

func (r *PostgresTripRepository) Get(ctx context.Context, id string) (model.Trip, error) {
	trip, err := scanTrip(r.db.QueryRowContext(ctx, `SELECT `+tripColumns+` FROM trips WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Trip{}, model.ErrTripNotFound
	}
	return trip, err
}

func (r *PostgresTripRepository) UpdateStatus(ctx context.Context, id string, status model.Status) (model.Trip, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Trip{}, err
	}
	defer tx.Rollback()

	var current model.Status
	if err := tx.QueryRowContext(ctx, `SELECT status FROM trips WHERE id = $1 FOR UPDATE`, id).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Trip{}, model.ErrTripNotFound
		}
		return model.Trip{}, err
	}
	if !model.CanTransition(current, status) {
		return model.Trip{}, model.ErrInvalidTransition
	}

	query := `UPDATE trips SET status = $2, updated_at = now() WHERE id = $1 RETURNING ` + tripColumns
	trip, err := scanTrip(tx.QueryRowContext(ctx, query, id, string(status)))
	if err != nil {
		return model.Trip{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Trip{}, err
	}
	return trip, nil
}

func (r *PostgresTripRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM trips WHERE id = $1`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return model.ErrTripNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanTrip(row rowScanner) (model.Trip, error) {
	var trip model.Trip
	var status string
	err := row.Scan(&trip.ID, &trip.TripNumber, &trip.Source, &trip.Destination, &trip.DriverName, &status, &trip.CreatedAt, &trip.UpdatedAt)
	trip.Status = model.Status(status)
	return trip, err
}

func isUniqueViolation(err error) bool {
	var pgError interface {
		error
		SQLState() string
	}
	return errors.As(err, &pgError) && pgError.SQLState() == "23505"
}
