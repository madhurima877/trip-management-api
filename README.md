# Trip Management API

A small Go 1.22 HTTP service backed by PostgreSQL. It supports trip creation, retrieval, paginated/filterable listing, status updates, and deletion.

## Run

```sh
docker-compose up -d postgres
go run ./cmd/server
```

Compose applies the migration when it creates the database volume for the first time. For an existing database, apply `migrations/001_create_trips.up.sql` manually. `DATABASE_URL` defaults to the local Compose database; `PORT` defaults to `8080`.

The API listens on `http://localhost:8080`.

## API

All request bodies are JSON. Unknown fields are rejected. Error responses use the same shape:

```json
{"error":{"code":"invalid_request","message":"..."}}
```

### Create a trip

`POST /trips` returns `201 Created`. Trip numbers are unique; a duplicate returns `409 Conflict`, making a repeated create unable to create a second trip.

```sh
curl -i -X POST http://localhost:8080/trips \
	-H 'Content-Type: application/json' \
	-d '{"trip_number":"TRIP-1001","source":"Boston","destination":"New York","driver_name":"Jordan Lee"}'
```

### List trips

`GET /trips?limit=20&offset=0&status=CREATED` returns `data` and pagination metadata. `limit` defaults to 20 and must be 1-100; `offset` defaults to 0. `status` is optional and must be one of `CREATED`, `ASSIGNED`, `IN_TRANSIT`, `COMPLETED`, or `CANCELLED`.

```sh
curl 'http://localhost:8080/trips?status=ASSIGNED&limit=10&offset=0'
```

### Get a trip

`GET /trips/{id}` accepts the UUID returned by create and returns `404` when it does not exist.

```sh
curl http://localhost:8080/trips/00000000-0000-0000-0000-000000000000
```

### Update status

`PATCH /trips/{id}/status` accepts `{"status":"ASSIGNED"}` and returns the updated trip. Allowed progress is `CREATED -> ASSIGNED -> IN_TRANSIT -> COMPLETED`; a trip may be cancelled from `CREATED`, `ASSIGNED`, or `IN_TRANSIT`. Invalid transitions, including moving backwards or changing a terminal state, return `409 Conflict`.

```sh
curl -i -X PATCH http://localhost:8080/trips/<trip-id>/status \
	-H 'Content-Type: application/json' \
	-d '{"status":"ASSIGNED"}'
```

Updates lock the row in a database transaction, check the current state, and update before releasing the lock. Concurrent updates are serialized and each is checked against the latest committed status.

### Delete a trip

`DELETE /trips/{id}` returns `204 No Content`; an unknown UUID returns `404`.

## Validation and status codes

- `400 Bad Request`: malformed or invalid input, unsupported query parameters, invalid UUID.
- `201 Created`: trip created.
- `200 OK`: trip retrieved, list returned, or status updated.
- `204 No Content`: trip deleted.
- `404 Not Found`: trip does not exist.
- `409 Conflict`: duplicate trip number or invalid status transition.
- `500 Internal Server Error`: unexpected server/database failure; implementation details are logged, not returned.

Required text fields are trimmed and validated for non-empty values and maximum lengths (trip number 100, other fields 255). PostgreSQL constraints enforce uniqueness, non-empty values, and valid statuses independently of the API.

## Design notes and follow-up

- A unique trip number is the idempotency boundary for create: retries with the same number receive `409` rather than creating duplicates. There is no separate `Idempotency-Key` store.
- Cancellation is allowed before completion; `COMPLETED` and `CANCELLED` are terminal.
- The migration is mounted as a PostgreSQL first-run initialization script in Compose. Existing persistent volumes do not rerun initialization scripts.
- With more time, add PostgreSQL integration tests for concurrent transactions and run migrations through a versioned migration runner for deployed environments.

## Tests

```sh
go test ./...
```
