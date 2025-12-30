.PHONY: build run test docker-build docker-up docker-down clean

# Build the application
build:
	go build -o bin/payment-api cmd/api/main.go

# Run the application locally
run:
	go run cmd/api/main.go

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# View test coverage
coverage:
	go tool cover -html=coverage.out

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Docker build
docker-build:
	docker build -t payment-api:latest .

# Start all services with docker-compose
docker-up:
	docker-compose up --build -d

# Stop all services
docker-down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out

# Load test
load-test:
	hey -n 10000 -c 100 -m POST \
		-H "Content-Type: application/json" \
		-d '{"user_id":"user1","merchant_id":"m1","amount":1000,"currency":"USD","method":"CREDIT_CARD"}' \
		http://localhost/api/v1/payments

# Database migrations (if using migrate tool)
migrate-up:
	migrate -path scripts -database "postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable" up

migrate-down:
	migrate -path scripts -database "postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable" down

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application locally"
	@echo "  test          - Run tests"
	@echo "  coverage      - View test coverage"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  deps          - Download dependencies"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start all services"
	@echo "  docker-down   - Stop all services"
	@echo "  logs          - View logs"
	@echo "  clean         - Clean build artifacts"
	@echo "  load-test     - Run load test"
