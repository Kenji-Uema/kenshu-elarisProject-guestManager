CONTAINER_TEST_PACKAGES := /integration_test($$|/)|/internal/infra/mdb$$|/internal/infra/mq$$
SAFE_TEST_PACKAGES := $(shell go list ./... | grep -vE '$(CONTAINER_TEST_PACKAGES)')

build: generate
	go build ./internal

test:
	go test -p 1 $(SAFE_TEST_PACKAGES)

test-unit:
	go test -p 1 $(SAFE_TEST_PACKAGES)

test-container:
	go test ./internal/infra/mdb ./internal/infra/mq

test-integration:
	go test ./integration_test/...

generate:
	npx buf generate

docker-build:
	 docker build --build-arg SERVICE_NAME=guest-manager --build-arg VERSION=latest -t guest-manager:latest .
