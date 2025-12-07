.PHONY: install-tools install-build-tools install-dev-tools \
	di deps lint gen gorm-gen mock test test-e2e up down-servers cli \
	migrate-file migrate-up migrate-down migrate-down-all \
	down docker-down volume-down format build \
	proto-setup proto-pull proto-gen proto-clean \
	grpc-server grpc-server-docker

## Init .env file
# .PHONY: init
# init: init-env install-tools install-dev-tools
# init-env:
# 	cp .env.example .env
	
install-tools: install-build-tools install-dev-tools
	
install-build-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/cmd/stringer@latest
	go install github.com/google/wire/cmd/wire@latest

install-dev-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

di:
	cd di && wire gen
	
deps:
	go get -u ./... 
	go mod download
	go mod tidy

lint:
	golangci-lint run

gen: gorm-gen
	go generate ./...

gorm-gen:
	go run _gorm_gen/main.go

mock:
	rm -rf ./mock
	go generate ./...

test: test-unit test-e2e vuln-check

test-unit: 
	ENVCODE=unit go test -v -race -coverprofile=cover.out $(shell go list ./... | grep -vE "(test|gen)/")
	@go tool cover -func=cover.out | grep "total:" | tr '\t' ' '
	go tool cover -html=cover.out -o cover.html

test-e2e:
	ENVCODE=e2e go test -v -race ./test/e2e/...

vuln-check:
	govulncheck ./...

cli:
	go run entrypoint/cli/main.go ${command}

migrate-file:
	migrate create -ext sql -dir sql/migrations -seq ${name}

migrate-up:
	migrate -path sql/migrations -database mysql://root:root@tcp\(localhost:3306\)/stock_price_repository up

migrate-down:
	migrate -path sql/migrations -database mysql://root:root@tcp\(localhost:3306\)/stock_price_repository down 1

migrate-down-all:
	migrate -path sql/migrations -database mysql://root:root@tcp\(localhost:3306\)/stock_price_repository down

down: docker-down volume-down

docker-down:
	docker compose down --volumes

volume-down:
	docker compose down --rmi all --volumes

format:
	npx sql-formatter@15.5.2 --fix  sql/_init_sql/create_database.sql

build:
	cd entrypoint/cli && GOARCH=arm GOOS=linux GOARM=7 go build -o spr-cli

up:
	@echo "Starting all services (db, redis, api, grpc-server) with Docker Compose..."
	docker compose up api grpc-server

api:
	@echo "Starting API server with hot reload on port 8080..."
	air -c .air.toml

# Proto definitions management
proto-setup:
	@if [ -d "proto-definitions" ]; then \
		echo "proto-definitions already exists. Use 'make proto-pull' to update."; \
	else \
		git clone https://github.com/Code0716/stock-price-proto.git proto-definitions; \
		echo "Proto definitions cloned successfully."; \
	fi

proto-pull:
	@if [ ! -d "proto-definitions" ]; then \
		echo "proto-definitions not found. Run 'make proto-setup' first."; \
		exit 1; \
	fi
	cd proto-definitions && git pull origin main

proto-gen:
	@if [ ! -d "proto-definitions" ]; then \
		echo "proto-definitions not found. Run 'make proto-setup' first."; \
		exit 1; \
	fi
	buf generate proto-definitions

proto-clean:
	rm -rf proto-definitions pb

# gRPC Server
grpc-server:
	@echo "Starting gRPC server with hot reload on port 50051..."
	air -c .air.grpc.toml

grpc-server-docker:
	@echo "Starting gRPC server with Docker Compose..."
	docker compose up grpc-server
