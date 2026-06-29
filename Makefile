.PHONY: help up down logs build backend-build backend-test backend-vet tidy migrate-status fmt seed-admin

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

up: ## Start the full stack (build + run) in the background
	docker compose up -d --build

edge: ## Start the stack with the Caddy TLS reverse proxy
	docker compose --profile edge up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail logs
	docker compose logs -f --tail=100

build: ## Build all images
	docker compose build

backend-build: ## Compile the Go backend
	cd backend && go build ./...

backend-vet: ## Vet the Go backend
	cd backend && go vet ./...

backend-test: ## Run backend tests
	cd backend && go test ./...

tidy: ## Tidy Go modules
	cd backend && go mod tidy

fmt: ## Format Go code
	cd backend && gofmt -w .

seed-admin: ## Promote a user to admin (ADMIN_EMAIL=you@example.com)
	docker compose exec -e ADMIN_EMAIL=$(ADMIN_EMAIL) backend /server || true
