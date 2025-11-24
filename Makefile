.PHONY: help setup start-temporal start-worker start-server test clean

help:
	@echo "Pizza Order DAG Demo - Makefile Commands"
	@echo ""
	@echo "Setup:"
	@echo "  make setup          - Install Go dependencies"
	@echo "  make start-temporal - Start Temporal server (Docker)"
	@echo ""
	@echo "Run:"
	@echo "  make start-worker   - Start Temporal worker"
	@echo "  make start-server   - Start HTTP API server"
	@echo "  make test          - Run test flow"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean         - Stop all services and clean up"

setup:
	go mod download
	@echo "Dependencies installed!"

start-temporal:
	docker-compose up -d
	@echo "Temporal server starting..."
	@echo "Temporal UI will be available at: http://localhost:8233"
	@echo "Waiting for Temporal to be ready..."
	@sleep 10

start-worker:
	go run worker/main.go

start-server:
	go run main.go

test:
	./test-flow.sh

clean:
	docker-compose down -v
	@echo "Temporal stopped and data cleaned"
