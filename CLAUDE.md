# karavantrack-api-server

Go 1.25 backend for YoolLive (cargo/load tracking). Layered architecture with CQRS-style usecases.

## Layers

`internal/delivery` (HTTP/WS/worker/consumers) → `internal/usecase/<domain>/{command,query}` → `internal/domain` (entities + repository interfaces) → `internal/infrastructure/persistence/{repository,cache}`.

Infrastructure wrappers with no domain knowledge live in `pkg/` (config, database/postgres, redis, nats, s3, smtp, firebase, security/JWT, wsrouter, otlp, logger).

## Conventions

- **One file = one usecase.** `internal/usecase/loads/command/assign.go` defines `AssignUsecase` + `NewAssignUsecase(...)` + the `AssignRequest`/`AssignResponse` DTOs, all in that one file. The domain's `usecase.go` aggregates usecases into `Command`/`Query` structs via embedding.
- **Usecase method shape:** start with `context.WithTimeout(ctx, u.contextDuration)`, then `otlp.Start(ctx, otel.Tracer("<domain>"), "<Name>", attribute...)` with `defer func(){ end(err) }()`, then a `var input struct{...}{ ... }` block that parses/validates raw input (uuid parsing etc.) and returns `inerr.NewErrValidation(...)` on failure.
- **Errors:** internal errors live in `internal/inerr`; they're mapped to HTTP/WS responses in `internal/delivery/outerr` (`outerr.BadRequest`/`Forbidden`/`Handle`, `*WS` variants for websocket).
- **Handlers split by audience:** `handlers/common` (public + any authenticated user), `handlers/shippers`, `handlers/carriers`. Each has its own `routes.go` with `RegisterRoutes(r, opts)`, wired into `internal/delivery/api/router.go` under `/api/v1`. Swagger is generated as two independent instances (carrier/shipper) via `make swagger-gen`.
- **Dependency wiring is manual** in `internal/app/server.go` (`ServerApp.Run`). Handlers receive everything through one struct, `delivery.HandlerOptions` (`internal/delivery/options.go`). Adding a usecase means: repo in `server.go`, usecase construction in `server.go`, field in `HandlerOptions`, assignment in the `opts := &delivery.HandlerOptions{...}` literal.
- **Database:** bun + pgx, Postgres only (no other SQL driver). Migrations via `golang-migrate`, SQL files in `migrations/` (`make migrate`, `make migrate-create`). No ORM models in the domain layer — mapping between domain entities and bun models happens in the repository.
- **Background jobs:** asynq (Redis) — task constructors in `internal/tasks`, handlers in `internal/delivery/worker/{router.go,handlers}`. The worker runs in the same process as the API (`go s.taskWorker.Run(mux)`). Separate one-shot binary `cmd/gps_notify` (cobra) finds loads without GPS for >15 min and pushes the driver.
- **Event bus:** NATS (`internal/service/broker`, events in `internal/events`, subscribers in `internal/delivery/consumers`) — used only for live GPS tracking.

## Commands

- `make build` / `make build-linux`
- `make test` — `go test -v -race -timeout 30s ./...`
- `make lint` — `golangci-lint run` + `go-arch-lint check`
- `make migrate` / `make migrate-create` / `make migrate-down` / `make migrate-force`
- `make swagger-gen`
