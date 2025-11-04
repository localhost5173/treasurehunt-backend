.PHONY: help build run test clean docker-up docker-down deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Download Go dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o treasurehunt main.go

run: ## Run the application
	go run main.go

test: ## Run tests
	go test -v ./...

test-api: ## Run API integration tests
	./test_api.sh

docker-up: ## Start MongoDB with Docker Compose
	docker-compose up -d

docker-down: ## Stop MongoDB Docker containers
	docker-compose down

docker-logs: ## View MongoDB logs
	docker-compose logs -f

clean: ## Clean build artifacts
	rm -f treasurehunt
	go clean

install: ## Install the application
	go install

dev: docker-up run ## Start MongoDB and run the application

all: deps build ## Download dependencies and build
