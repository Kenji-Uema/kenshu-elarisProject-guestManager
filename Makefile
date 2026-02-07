build: generate
	go build .

generate:
	npx buf generate

docker-build:
	 docker build --build-arg SERVICE_NAME=guest-manager --build-arg VERSION=latest -t guest-manager:latest .