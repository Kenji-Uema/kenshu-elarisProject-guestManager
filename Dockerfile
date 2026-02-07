FROM golang:1.25.6-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/guestManager ./

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=build /out/guestManager /guestManager

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/guestManager"]
