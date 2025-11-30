# insider-assessment
### Overview

This repository contains my solution for the **Insider Assessment Project**. 

This project is a small but production-inspired messaging service built for an assessment scenario.

- Pulling **pending** messages in batches on a fixed schedule,
- Processing them through a **concurrent worker pool**,
- Delivering each message to an external **webhook-based SMS provider**,
- Updating the delivery status (`PENDING → SUCCESS / FAILED`) and timestamps,
- Exposing a **REST API** to control the scheduler (start/stop) and to list sent messages with pagination.

The codebase is intentionally structured with clear layers:

- A **domain layer** (`Message`, `Status`, validation rules) that is independent of frameworks,
- A **repository layer** using GORM for persistence (swappable with sqlc or similar tools),
- A **service layer** that encapsulates business logic and batch processing,
- A **scheduler** component that safely coordinates periodic execution and graceful shutdown,
- A thin **HTTP layer** (handlers, router, middleware) exposing a JSON API and Swagger documentation.

Although this is a prototype, the design aims to demonstrate real-world patterns:
separation of concerns, safe concurrency with contexts, and configuration-driven behavior
for intervals, batch sizes, timeouts, and worker counts.

---
### Message Lifecycle (end-to-end)
```text
[New message row]
        ↓
[messages table: PENDING]
        ↓
[Scheduler tick (SCHEDULER_INTERVAL)]
        ↓
[MessageService.ProcessBatch]
        ↓
[Worker pool (concurrent workers)]
        ↓
[sms.Client (Webhook)]
        ↓
[Update DB status: SUCCESS / FAILED]
        ↓
[cache in Redis by external messageId]
        ↓
[Exposed via GET /messages/sent]
```
---

### Architecture & Design
At a high level:

- **Domain** layer models the core business concepts (`Message`, `Status`) and rules.
- **Repository** layer handles persistence and database-specific concerns.
- **Service layer** orchestrates use-cases (batch processing, worker pool).
- **Scheduler** triggers batch processing on a fixed interval with graceful start/stop.
- **HTTP API** exposes control and query endpoints.
- **Infrastructure** (Postgres, Redis, webhook-based SMS provider) is kept behind interfaces.

The wiring of all dependencies happens in `cmd/api/main.go`, which acts as the composition root of the application.

---

###  Code Structure
The most important packages are:
- `internal/domain/message`
  - Contains the `Message` entity and `Status` enum.
  - Enforces invariants in the constructor `NewMessage` (non-empty recipient, non-empty content, max content length).
  - Defines the `Repository` interface; the domain layer does not know anything about GORM or SQL.
- `internal/repository/gorm/message`
  - GORM-based implementation of `message.Repository`.
  - MessageModel is the database model with indexes for sent_at, message_id, and created_at.
  - Uses SELECT … FOR UPDATE SKIP LOCKED to safely lock “pending” rows when multiple workers/processes are running.
  - Provides mapping helpers (toDomain, fromDomain) so the rest of the code always works with domain types.
- `internal/service`
  - `MessageService` implements higher-level operations on messages (e.g. `ProcessBatch`, `GetSent`).
  - Encapsulates the worker pool used to process messages concurrently.
  - Uses the domain model methods (`MarkSent`, `MarkFailed`) and then persists via `message.Repository`.
- `internal/scheduler`
  - `SchedulerService` periodically calls `BatchProcessor.ProcessBatch(ctx)` with a configurable interval and batch timeout.
  - Maintains its own internal event loop and state (`running`, `inBatch`, `pendingStop`) without external locks.
  - Exposes a small control surface: `Start()`, `Stop()`, and `IsRunning()`.
- `internal/handler`, `internal/router`, `internal/server`
  - `handler` contains HTTP handlers (home/health, scheduler start/stop, sent messages listing).
  - `router` registers routes with their handlers.
  - `server` wraps `http.Server` and applies middleware (`RequestLogger`) through a simple `Chain` function.
- `internal/cache/redis` and `internal/sms`
  - Redis cache adapter used to store sent message metadata keyed by external message ID.
  - `sms.Client` interface plus a `WebhookClient` implementation that sends JSON to a webhook endpoint and parses the response.

---

###  Concurrency: Scheduler & Worker Pool

Message sending is split into two distinct concurrent components: the scheduler and the worker pool.

#### Scheduler

The scheduler is responsible for when to run a batch:
- `SchedulerService` runs a dedicated goroutine with an internal loop.
- It uses a time.Ticker to fire every `SCHEDULER_INTERVAL` (e.g. 5s).
- On each tick, if the scheduler is `running` and no batch is in progress, it calls:
```` 
     ctx, cancel := context.WithTimeout(context.Background(), batchTimeout)
     err := messageService.ProcessBatch(ctx)
     cancel()
````
- `Start()` and `Stop()` are synchronous:
  - `Start()` marks the scheduler as running and returns once the internal loop has acknowledged the state.
  - `Stop()` waits until the currently running batch (if any) completes or times out before returning.

This design allows the scheduler to be started and stopped via the HTTP API without race conditions or partial shutdowns.


#### Service-level Worker Pool
The worker pool is responsible for how batches are processed:
- ProcessBatch:
  - Fetches up to MESSAGE_BATCH_SIZE pending messages in a single query.
  - Decides how many workers to start, up to `MESSAGE_MAX_WORKERS`.
  - Spawns workers that each process a “stride” of messages:
    - worker 1: `indices 0, 4, 8, ...`
    - worker 2: `indices 1, 5, 9, ...`
    - etc.
  - For each message:
    - A per-message context is derived:
    ```` 
    msgCtx, cancel := context.WithTimeout(ctx, MESSAGE_PER_MESSAGE_TIMEOUT)
    ````
    - The SMS is sent via sms.Client.Send.
    - The domain entity is updated with `MarkSent` or `MarkFailed`, and the new state is persisted via `UpdateStatus`.
    - A `sync.WaitGroup` ensures the batch is fully processed before returning.
    - If the parent context is cancelled (e.g. because the scheduler’s batch timeout was exceeded), workers stop processing new messages and exit gracefully.

This separation of concerns keeps timing, retries, and parallelism local to the service, while the scheduler only deals with intervals and lifecycle.

---

### Persistence: GORM & Trade-offs

This project uses GORM for database access as a pragmatic choice for a prototype.

#### Why GORM here?

- Very quick to set up and iterate with.
- Model definitions and simple queries are concise and easy to read.
- Includes features like:
    - AutoMigrate for the `messages` table in the seeding command.
    - Locking hints (`Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})`) to avoid double-processing rows.

#### Why not sqlc or similar tools?

In a production environment, tools like **sqlc** (or other code-generating SQL mappers) are often preferable because:

- They provide compile-time guarantees that SQL and Go types are in sync.
- They tend to be more explicit and can be more performant.
- They make complex queries easier to reason about at the SQL level.

For a time-boxed assessment / prototype, the overhead of introducing sqlc (schema management, generation step, extra tooling) is not strictly necessary. GORM gives enough ergonomics to focus on design, concurrency patterns, and API structure.

#### Connection pooling & tuning (future work)

The current GORM setup intentionally keeps the DB configuration minimal.  
In a real production deployment, we would also:

- Explicitly configure the underlying `sql.DB` pool via:
    - `SetMaxOpenConns` (max concurrent connections),
    - `SetMaxIdleConns` (idle connections to keep around),
    - `SetConnMaxLifetime` (lifetime before a connection is recycled).
- Drive these values from configuration (e.g. `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`) so they can be tuned per environment.
- Align pool limits with Postgres settings and the scheduler/worker configuration (number of workers, batch sizes), to avoid overloading the database under high load.

For the scope of this assessment, the default GORM pool is acceptable; however, explicit connection-pool tuning would be one of the first improvements for a production-grade version of this service.

---

### Pagination Strategy
The sent-messages endpoint uses simple page/limit pagination:
- Input parameters:
  - `page` (1-based, default 1)
  - `limit` (default 20, capped to a maximum)
- Implementation:
  - Only messages with `Status = SUCCESS` are included.
  - Ordered by `sent_at DESC` (most recent first).
  - Performed with `LIMIT` + `OFFSET` and a separate `COUNT(*)` to return the total number of records.

This approach is:
- Very easy to consume from a client.
- Simple to test and reason about.
- Sufficient for demonstrating listing and pagination in this project.

#### Alternative: Cursor-based Pagination (Not Implemented)
In larger-scale systems with very large tables, cursor-based or seek-based pagination can be a better choice:

- Uses a stable ordering key `(sent_at, id, etc.)` and a “cursor” like `lastSeenSentAt`.
- Avoids large `OFFSET` values, which can be inefficient.
- More resilient to new rows being inserted while the client is paging.

For this exercise, page/limit was chosen for its simplicity and readability, but the repository structure makes it straightforward to swap in a cursor-based approach later.

---

### Environment Variables
The service is configured via a `.env` file.

Example `.env` file:

```env
# App
APP_NAME=insider-assessment
APP_ENV=development          # development | production

# API Server
API_HOST=127.0.0.1
API_PORT=8080             # docker-compose port: "8080:8080"

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Postgresql
DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=123456
DB_NAME=db_ins_message
DB_SSLMODE=disable

# SMS Service
SMS_PROVIDER_URL=https://webhook.site/d4de976a-6ef9-4ede-96e5-57be6c4b1467
SMS_PROVIDER_KEY=INS.me1x9uMcyYGlhKKQVPoc.bO3j9aZwRTOcA2Ywo

# Scheduler
SCHEDULER_INTERVAL=2m          
SCHEDULER_BATCH_TIMEOUT=10s    

# Message Process
MESSAGE_BATCH_SIZE=2           
MESSAGE_MAX_WORKERS=2          
MESSAGE_PER_MESSAGE_TIMEOUT=5s
```
---


### Run with Docker Compose
```bash
docker-compose up --build
```
This will start:
- Postgresql (with `messages` table)
- Redis
- Seed command
- API service on `localhost` with port number that you set in `.env`

Once the stack is up, you can:
examples:
- Check health: `GET http://localhost:8080/health`
- Ping the API: `GET http://localhost:8080/ping`
- Open Swagger UI in the browser:`http://localhost:8080/swagger/`

## Future Improvements

This project is meant as an assessment-friendly prototype, but it’s structured so it can grow.

Some natural next steps:

- **Concurrency & throughput**
    - Make `MESSAGE_BATCH_SIZE` and `MESSAGE_MAX_WORKERS` adaptive (based on queue size, provider latency, error rate).
    - Add better coordination when multiple instances run in parallel (leader election or distributed locking).

- **Reliability**
    - Introduce a **Dead Letter Queue (DLQ)** for messages that keep failing.
    - Add retries with exponential backoff instead of a single attempt per scheduler run.

- **Observability**
    - Export metrics (batch size/duration, success/failure counts, provider latency) to Prometheus / OpenTelemetry.
    - Switch to structured logging with correlation/message IDs for easier debugging.

- **Persistence & schema**
    - Tune the DB connection pool via config (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`).
    - Add more targeted indexes or partitioning if the `messages` table grows large.

- **API & UX**
    - Add filters on `GET /messages/sent` (date range, recipient, status).
    - Optionally add cursor-based pagination alongside page/limit.

### Testing

Right now, most of the effort went into the architecture and end-to-end flow; test coverage is intentionally minimal.

If I had more time, I’d add:

- Unit tests for `MessageService` and HTTP handlers (`httptest`),
- Repository tests against a test database,
- A small integration test that:
    - seeds `PENDING` messages,
    - runs the scheduler,
    - and asserts they appear as `SUCCESS` in `GET /messages/sent`.

The good part is: the current design (interfaces for repository, SMS client, scheduler) already makes this straightforward to add later.

## Conclusion

This is a small project on purpose, but I wanted it to feel like a real service rather than a throwaway script.

The core flow is fully implemented:

`PENDING row in Postgres → scheduled batch → concurrent worker pool → webhook SMS provider → updated status → exposed via API`

There are plenty of things that could be added on top (DLQs, retries, metrics, more tests), but the foundation is in place and easy to extend. That was the main goal for this assessment.

