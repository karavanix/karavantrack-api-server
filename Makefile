-include .env
export

CURRENT_DIR=$(shell pwd)
APP=yoollive-api-server
CMD_DIR=./cmd


# build and test
.PHONY: build build-linux test
build:
	go build -o ./bin/${APP} ${CMD_DIR}/main.go

build-linux:
	CGO_ENABLED=0 GOARCH="amd64" GOOS=linux go build -ldflags="-s -w" -o ./bin/${APP} ${CMD_DIR}/main.go
test:
	go test -v -race -timeout 30s ./...

# run linters
.PHONY: lint
lint:
	golangci-lint run
	go-arch-lint check

# migrate
.PHONY: migrate migrate-down migrate-create migrate-force
migrate:
	migrate -path ./migrations -database "postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_DATABASE}?sslmode=disable" up
migrate-down:
	migrate -path ./migrations -database "postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_DATABASE}?sslmode=disable" down
migrate-create:
	@read -p "Enter migration name (e.g. table_name): " name; \
	migrate create -ext sql -dir ./migrations -seq $$name
migrate-force:
	@read -p "Enter the migration version to force: " version; \
	migrate -path ./migrations -database  "postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_DATABASE}?sslmode=disable" force $$version

# generate swagger
.PHONY: swagger-gen swagger-gen-carrier swagger-gen-shipper
swagger-gen: swagger-gen-carrier swagger-gen-shipper

swagger-gen-carrier:
	swag init --instanceName carrier --parseDependency --dir ./internal/delivery/api/handlers/carriers,./internal/delivery/api/handlers/common -g routes.go -o ./internal/delivery/api/docs/carrier

swagger-gen-shipper:
	swag init --instanceName shipper --parseDependency --dir ./internal/delivery/api/handlers/shippers,./internal/delivery/api/handlers/common -g routes.go -o ./internal/delivery/api/docs/shipper

.PHONY: asynqmon
asynqmon:
	docker run -d --rm --name asynqmon --network host hibiken/asynqmon --redis-addr=localhost:6379

.DEFAULT_GOAL := build