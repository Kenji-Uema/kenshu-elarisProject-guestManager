# guestManager

Owns guest-facing state and the lodging WebSocket flow.

## Responsibilities

- expose guest HTTP APIs
- manage lodging chat over WebSocket
- coordinate check-in, stay, and checkout interactions
- react to simulated time for breakfast, dinner, and checkout notifications
- integrate guest, booking, cottage, cleaning, and cache concerns

## Interfaces

- HTTP API for guest operations
- WebSocket lodging chat endpoint
- gRPC client to `clockSimulator`
- RabbitMQ producer for cleaning requests
- MongoDB and Redis persistence

## Run

```sh
go run .
```

## Build

```sh
make build
make docker-build/home/kenjiuema/Documents/projects/guestManager/internal/infra/mq/fakes
```

## Configuration/home/kenjiuema/Documents/projects/guestManager/internal/infra/mq/.testcontainers-tmp-2822467785/home/kenjiuema/Documents/projects/guestManager/internal/infra/mq/rabbitmq_test.go

Configuration is environment-driven. See:

- `internal/config/config.go`
- `internal/config/rabbitmq_config.go`
  /home/kenjiuema/Documents/projects/guestManager/internal/infra/mq/.testcontainers-tmp-2822467785
Important families:

- service HTTP: `SERVICE_*`
- MongoDB: `MONGO_*`
- Redis: `REDIS_*`
- RabbitMQ: `RABBITMQ_*`, `CLEANING_EXCHANGE_*`
- clock client: `CLOCK_EMU_GRPC_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Entry points

- `main.go`
- `internal/main.go`
- `internal/app/reception_service.go`
- `internal/transport/websocket/`
