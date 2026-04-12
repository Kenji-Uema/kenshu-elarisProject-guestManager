# Guest Manager

Owns guest records and the guest-facing lodging interaction API.

## Main Docs

See the main project documentation: <https://kenji-uema.github.io/kenshu-elarisProject-docs/>

## What It Does

- exposes guest HTTP endpoints for registration and guest lookup
- serves the lodging WebSocket session used during check-in, stay, and checkout
- publishes guest-facing journey notifications and cleaning requests
- coordinates cottage, booking, cleaning, cache, and clock-driven lodging behavior
- uses Redis as short-lived runtime state during lodging interactions

## Interfaces

- HTTP: `/guest`, `/guest/:userId`, `/guest/:userId/bookings`
- WebSocket: `/lodging/chat`
- probes: `/healthz`, `/readyz`
- RabbitMQ publishers for guest communication and cleaning requests
- RabbitMQ consumers for day-change and hour-change events
- gRPC client to `clockSimulator`

## RabbitMQ Specification

Produces:

- exchange `ex.cleaning.request` via `CLEANING_EXCHANGE_NAME`
  routing key: empty string `""`
  exchange kind: `direct` by default via `CLEANING_EXCHANGE_KIND`
  payloads: protobuf cleaning requests published by the cleaning service
  message example:
  ```json
  {
    "roomName": "Barbara Karst",
    "request": "PREPARE_FOR_GUEST"
  }
  ```
- exchange `ex.communication` via `GUEST_COMMUNICATION_EXCHANGE_NAME`
  routing key: `guest.<guestId>`
  exchange kind: `direct` by default via `GUEST_COMMUNICATION_EXCHANGE_KIND`
  payloads: protobuf guest communication messages, currently check-in-today notifications
  message example:
  ```json
  {
    "booking_id": "69d56ce351d7f8bbb753b70b",
    "guest_id": "69d56ce3543361444126758c",
    "cottage_name": "Barbara Karst",
    "check_in": "2025-04-15T00:00:00Z",
    "check_out": "2025-04-18T00:00:00Z",
    "number_of_guests": 1,
    "notification_day": "2025-04-14T00:00:00Z"
  }
  ```

Consumes:

- queue `q.guest-manager.day-change` via `DAY_CHANGE_QUEUE_NAME`
  bound to exchange `ex.time.event` via `DAY_CHANGE_BINDING_EXCHANGE_NAME`
  binding key: `time.event.day` via `DAY_CHANGE_BINDING_ROUTING_KEY`
  purpose: receives day-change time events from the clock/event publisher
  message example:
  ```json
  {
    "time": "2025-04-14T00:00:00Z"
  }
  ```
- queue `q.guest-manager.hour-change` via `HOUR_CHANGE_QUEUE_NAME`
  bound to exchange `ex.time.event` via `HOUR_CHANGE_BINDING_EXCHANGE_NAME`
  binding key: `time.event.hour` via `HOUR_CHANGE_BINDING_ROUTING_KEY`
  purpose: receives hour-change time events from the clock/event publisher
  message example:
  ```json
  {
    "time": "2025-04-14T13:00:00Z"
  }
  ```

Notes:

- queue and exchange names above are the current defaults from configuration; all are environment-driven
- produced messages use protobuf with AMQP content type `application/protobuf`
- JSON snippets above are human-readable representations of the protobuf payloads, not the wire format itself
- consumers use manual acknowledgements; invalid time-event payloads are nacked without requeue

## WebSocket Specification

Endpoint:

- `GET /lodging/chat`

Transport:

- upgrades an HTTP request to a WebSocket connection
- uses text frames containing JSON-encoded `lodging.v1.ChatMessage` envelopes
- protocol version used by the server is `lodging.v1`
- current upgrader origin policy accepts any origin

Envelope shape:

```json
{
  "messageId": "6df1f733-4cb2-4dc4-a842-75f7bcd2a437",
  "correlationId": "6df1f733-4cb2-4dc4-a842-75f7bcd2a437",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "guestAction": "SHOW_FOR_CHECKIN"
}
```

Common fields:

- `messageId`: required unique message id
- `correlationId`: links replies to the triggering message
- `sender`: `SENDER_GUEST` or `SENDER_SYSTEM`
- `protocolVersion`: expected to be `lodging.v1`
- `traceContext`: optional tracing metadata map
- exactly one payload variant is set per message: `guestAction`, `guestResponse`, `systemNotification`, `systemRequest`, or `ack`

Acknowledgement rules:

- when the server receives a guest message, it automatically replies with an `ack`
- when the server sends a `systemRequest` or `systemNotification`, the client is expected to reply with an `ack`
- `ack.acknowledgedMessageId` must contain the message id being acknowledged
- `systemRequest` messages are retried by the server if an ack is not received within the ack timeout

Ack example:

```json
{
  "messageId": "f0f3da31-02d2-48d2-a807-bad65c69dfd5",
  "correlationId": "6df1f733-4cb2-4dc4-a842-75f7bcd2a437",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "ack": {
    "acknowledgedMessageId": "6df1f733-4cb2-4dc4-a842-75f7bcd2a437",
    "status": "ACK_STATUS_ACCEPTED",
    "code": "ERROR_CODE_NONE"
  }
}
```

System requests emitted by the server:

- `REQUEST_DOCUMENT`
- `REQUEST_BOOKING_NUMBER`
- `GIVE_COTTAGE_KEY`
- `REQUEST_COTTAGE_KEY`

System notifications emitted by the server:

- `BOOKING_CHECKING`
- `CHECK_IN_COMPLETE`
- `DINNER_READY`
- `BREAKFAST_READY`
- `CHECK_OUT_COMPLETE`

Guest actions accepted by the server during the normal flow:

- `SHOW_FOR_CHECKIN`
- `TAKE_COTTAGE_KEY`
- `ENTER_COTTAGE`
- `GO_FOR_A_BATH`
- `GO_FOR_DINNER`
- `GO_TO_SLEEP`
- `WAKEUP`
- `GO_FOR_BREAKFAST`
- `LEAVE_CLEANUP_NOTIFICATION`
- `ENJOY_RESORT`
- `LEAVE_COTTAGE`
- `PROCEED_TO_CHECKOUT`
- `RETURN_COTTAGE_KEY`

Guest response payload examples:

Document response:

```json
{
  "messageId": "1b5f9cf6-e0e9-44c0-99d6-95508f59daf8",
  "correlationId": "server-request-message-id",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "guestResponse": {
    "showDocument": {
      "documentId": "ID-001"
    }
  }
}
```

Booking number response:

```json
{
  "messageId": "8979db13-74f6-4bd5-a461-8d8f48e63063",
  "correlationId": "server-request-message-id",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "guestResponse": {
    "showBookingNumber": {
      "bookingId": "64b6f7c2c0f1e84c0a1a9d01"
    }
  }
}
```

Receive key response:

```json
{
  "messageId": "5560ca7a-3c3e-4f48-b0f7-a3e023da5896",
  "correlationId": "server-request-message-id",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "guestResponse": {
    "receiveCottageKey": {
      "cottageKeyId": "lake-house-key-1"
    }
  }
}
```

Return key response:

```json
{
  "messageId": "4c7666a0-1600-4553-8dae-f849ff1daf8b",
  "correlationId": "server-request-message-id",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "guestResponse": {
    "returnCottageKey": {
      "cottageKeyId": "lake-house-key-1"
    }
  }
}
```

Success-path session sequence:

1. guest sends `SHOW_FOR_CHECKIN`
2. server sends `REQUEST_DOCUMENT`
3. guest sends `guestResponse.showDocument`
4. server sends `BOOKING_CHECKING`
5. server sends `CHECK_IN_COMPLETE`
6. server sends `GIVE_COTTAGE_KEY`
7. guest sends `guestResponse.receiveCottageKey`
8. guest sends `TAKE_COTTAGE_KEY`
9. guest proceeds through stay actions such as `ENTER_COTTAGE`, `GO_FOR_A_BATH`, `GO_FOR_DINNER`, `GO_TO_SLEEP`, `WAKEUP`, `GO_FOR_BREAKFAST`, `LEAVE_CLEANUP_NOTIFICATION`, and `ENJOY_RESORT`
10. server emits timed notifications such as `BREAKFAST_READY` and `DINNER_READY`
11. guest sends `PROCEED_TO_CHECKOUT`
12. server sends `REQUEST_COTTAGE_KEY`
13. guest sends `guestResponse.returnCottageKey`
14. guest sends `RETURN_COTTAGE_KEY`
15. server sends `CHECK_OUT_COMPLETE`

Notes:

- the websocket session runs check-in, then stay, then checkout in a single connection
- all examples above are success-path examples
- message examples are JSON representations of the protobuf-generated envelope used by the server

## HTTP Examples

All examples in this section are success-path samples and are expected to return `2xx` responses when the service dependencies are healthy and the referenced data exists.

Set a base URL once:

```sh
BASE_URL=http://localhost:8080
```

Create a guest:

```sh
curl -i \
  -X POST "$BASE_URL/guest" \
  -H 'Content-Type: application/json' \
  -d '{
    "document_id": "208176428",
    "given_names": "Camryn",
    "surname": "Wiggins",
    "email": "Camryn.Wiggins@test.com",
    "billing_address": "6074 East Damside, Philadelphia, New Hampshire 29823"
  }'
```

Status code: `201 Created`

Example response:

```json
"69d555b35433614441267552"
```

Fetch a guest by id:

```sh
curl -i "$BASE_URL/guest/69d555b35433614441267552"
```

Status code: `200 OK`

Example response:

```json
{
  "document_id": "208176428",
  "given_names": "Camryn",
  "surname": "Wiggins",
  "email": "Camryn.Wiggins@test.com",
  "billing_address": "6074 East Damside, Philadelphia, New Hampshire 29823",
  "created_at": "2025-01-01T19:43:55Z"
}
```

Update a guest:

```sh
curl -i \
  -X PATCH "$BASE_URL/guest/69d555b35433614441267552" \
  -H 'Content-Type: application/json' \
  -d '{
    "document_id": "208176428",
    "given_names": "Camryn Elise",
    "surname": "Wiggins",
    "email": "camryn.wiggins@example.com",
    "billing_address": "6074 East Damside, Philadelphia, New Hampshire 29823",
    "created_at": "2025-01-01T19:43:55Z"
  }'
```

`created_at` is optional on update. If omitted, the handler fills it with the current clock time.

Status code: `200 OK`

Example response:

```json
{
  "document_id": "208176428",
  "given_names": "Camryn Elise",
  "surname": "Wiggins",
  "email": "camryn.wiggins@example.com",
  "billing_address": "6074 East Damside, Philadelphia, New Hampshire 29823",
  "created_at": "2025-01-01T19:43:55Z",
  "last_update": "2025-01-01T20:10:00Z"
}
```

Fetch bookings for a guest:

```sh
curl -i "$BASE_URL/guest/69d555b45433614441267554/bookings"
```

Status code: `200 OK`

Example response:

```json
[
  {
    "number_of_guests": 1,
    "stay_period": {
      "checkin": "2025-01-06T00:00:00Z",
      "checkout": "2025-01-09T00:00:00Z"
    },
    "cottage_name": "Barbara Karst",
    "status": "confirmed"
  }
]
```

Liveness and readiness probes:

```sh
curl -i "$BASE_URL/healthz"
curl -i "$BASE_URL/readyz"
```

`GET /healthz`
Status code: `200 OK`
Example response: empty body

`GET /readyz`
Status code: `200 OK`
Example response: empty body

## Local Commands

```sh
go run ./internal
go build ./internal
make test
make test-unit
make test-container
make test-integration
make generate
make docker-build
```

`make test` and `make test-unit` run the safe default suite and skip container-heavy tests.
`make test-container` and `make test-integration` require Docker.

## Minimum Env To Start

Optional vars with defaults, such as `SERVICE_NAME`, collection names, exchange durability flags, Redis DB, and timeout values, are omitted here.

```sh
SERVICE_HOST=0.0.0.0
SERVICE_PORT=8080

CLOCK_EMU_GRPC_HOST=<clock host>
CLOCK_EMU_GRPC_PORT=50051

MONGO_INITDB_ROOT_USERNAME=<mongo user>
MONGO_INITDB_ROOT_PASSWORD=<mongo password>
MONGO_HOST=<mongo host>
MONGO_DATABASE=cottages

RABBITMQ_USERNAME=<rabbit user>
RABBITMQ_PASSWORD=<rabbit password>
RABBITMQ_HOST=<rabbit host>
RABBITMQ_PORT=5672

REDIS_HOST=<redis host>
REDIS_PORT=6379

CLEANING_EXCHANGE_NAME=ex.cleaning.request
CLEANING_EXCHANGE_KIND=direct
GUEST_COMMUNICATION_EXCHANGE_NAME=ex.communication
GUEST_COMMUNICATION_EXCHANGE_KIND=direct

DAY_CHANGE_QUEUE_NAME=q.guest-manager.day-change
DAY_CHANGE_BINDING_EXCHANGE_NAME=ex.time.event
DAY_CHANGE_BINDING_ROUTING_KEY=time.event.day

HOUR_CHANGE_QUEUE_NAME=q.guest-manager.hour-change
HOUR_CHANGE_BINDING_EXCHANGE_NAME=ex.time.event
HOUR_CHANGE_BINDING_ROUTING_KEY=time.event.hour

OTEL_EXPORTER_OTLP_ENDPOINT=<otel host>
OTEL_EXPORTER_OTLP_GRPC_PORT=4317
OTEL_EXPORTER_OTLP_HEALTH_PORT=13133
OTEL_EXPORTER_OTLP_INSECURE=true
```

## Configuration

Configuration is environment-driven. Start with:

- `internal/config/config.go`
- `internal/config/rabbitmq_config.go`

Important groups:

- service HTTP and timeout settings: `SERVICE_*`
- MongoDB: `MONGO_*`
- Redis: `REDIS_*`
- RabbitMQ connection plus publisher/consumer settings: `RABBITMQ_*`, `CLEANING_*`
- clock client: `CLOCK_EMU_GRPC_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Key Files

- `internal/main.go`
- `internal/app/reception_service.go`
- `internal/transport/init.go`
- `internal/transport/websocket/`
- `integration_test/`
