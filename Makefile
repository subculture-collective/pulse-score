.PHONY: build run test lint clean migrate-up migrate-down migrate-create dev-db dev-db-down

BINARY_NAME=pulsescore-api
BUILD_DIR=bin
DATABASE_URL ?= postgres://pulsescore:pulsescore@localhost:5432/pulsescore_dev?sslmode=disable

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test ./... -v -race

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=my_migration"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(NAME)

dev-db:
	docker compose -f docker-compose.dev.yml up -d

dev-db-down:
	docker compose -f docker-compose.dev.yml down
