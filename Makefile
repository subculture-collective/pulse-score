.PHONY: build run test lint clean migrate-up migrate-down

BINARY_NAME=pulsescore-api
BUILD_DIR=bin

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
	@echo "Run migrations up (requires golang-migrate)"
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "Run migrations down (requires golang-migrate)"
	migrate -path migrations -database "$(DATABASE_URL)" down

dev-db:
	docker compose -f docker-compose.dev.yml up -d

dev-db-down:
	docker compose -f docker-compose.dev.yml down
