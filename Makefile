.PHONY: install-tools install-build-tools install-dev-tools \
	di deps lint gen gorm-gen mock test test-e2e up cli \
	migrate-file migrate-up migrate-down migrate-down-all \
	down docker-down volume-down format build

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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest

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

test:  test-unit  test-e2e

test-unit: 
	ENVCODE=unit go test -v -race -coverprofile=cover.out $(shell go list ./... | grep -vE "(test|gen)/")
	@go tool cover -func=cover.out | grep "total:" | tr '\t' ' '
	go tool cover -html=cover.out -o cover.html

test-e2e:
	ENVCODE=e2e go test -v -race ./test/e2e/...

up:
	docker compose up -d
	# air -c .air.toml

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
