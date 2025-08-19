SHELL := /bin/bash

.PHONY: run-app run-consumer run-producer tidy deps lint test test-unit test-repo migrate-install migrate-create migrate-up migrate-down build up down

export GO111MODULE=on

run-app:
	go run ./cmd/app


build:
	go build -o bin/app ./cmd/app

tidy:
	go mod tidy

deps:
	go get \
		github.com/gin-gonic/gin \
		github.com/segmentio/kafka-go \
		github.com/jackc/pgx/v5/pgxpool \
		go.uber.org/zap \
		github.com/kelseyhightower/envconfig \
		github.com/stretchr/testify

test:
	go test -v ./...

migrate-install:
	@echo "Installing golang-migrate CLI..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Installed. Make sure GOPATH/bin is in PATH (e.g. export PATH=\"$$HOME/go/bin:$$PATH\")."
	@export PATH="$$($(go env GOPATH))/bin:$$PATH"; migrate -version | cat || true

migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=<migration_name>" && exit 1)
	@PATH="$(shell go env GOPATH)/bin:$$PATH" migrate create -ext sql -dir ./migrations -seq $(name)

migrate-up:
	@migrate -path ./migrations -database "$$PG_DSN" up

migrate-down:
	@migrate -path ./migrations -database "$$PG_DSN" down 1
