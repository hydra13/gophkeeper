.PHONY: fmt lint test test-unit test-integration test-e2e cover cover-check proto proto-check build build-server build-client build-client-cli build-client-tui build-client-desktop build-client-web dev-client-web clean dev-up dev-down dev-reset test-storage-integration

PROTO_SRC := rpc/proto/v1/*.proto
PROTO_OUT := internal/rpc/pbv1
MODULE := github.com/hydra13/gophkeeper
INTEGRATION_PACKAGES := ./internal/jobs/reencrypt ./internal/repositories/database ./internal/services/sync ./internal/services/uploads ./internal/storage
E2E_PACKAGES := ./tests/e2e
GO_FILES := $(shell git ls-files '*.go')

fmt:
	goimports -w $(GO_FILES)

lint:
	golangci-lint run

test-unit:
	go test -race ./...

test:
	@PACKAGES="$$(bash scripts/list-cover-packages.sh lines)" && \
	COVERPKG="$$(bash scripts/list-cover-packages.sh csv)" && \
	LOG_FILE=coverage_test.log && \
	if ! go test -coverprofile=coverage.out -coverpkg="$$COVERPKG" $$PACKAGES >"$$LOG_FILE" 2>&1; then \
		tail -n 50 "$$LOG_FILE"; \
		exit 1; \
	fi && \
	rm -f "$$LOG_FILE"

test-integration:
	go test -p 1 -tags=integration -count=1 $(INTEGRATION_PACKAGES)

test-e2e:
	go test -tags=e2e -count=1 $(E2E_PACKAGES)

cover: test
	go tool cover -html=coverage.out

cover-check: test
	@grep -ve '/mocks/' -e '\.pb\.go' -e '/proto/v1/' -e '/pbv1/' coverage.out > coverage_filtered.out
	@COVERAGE=$$(go tool cover -func=coverage_filtered.out | tail -1 | awk '{print $$NF}' | tr -d '%') && \
	rm -f coverage_filtered.out && \
	echo "Coverage (excl. mocks, generated): $${COVERAGE}%" && \
	if [ "$$(echo "$$COVERAGE < 70" | bc -l)" -eq 1 ]; then \
		echo "FAIL: coverage $${COVERAGE}% is below 70% threshold"; \
		exit 1; \
	fi && \
	echo "PASS: coverage $${COVERAGE}% meets 70% threshold"

proto:
	@mkdir -p $(PROTO_OUT)
	@find $(PROTO_OUT) -maxdepth 1 -name '*.pb.go' -delete
	protoc -I . \
	       --go_out=. --go_opt=module=$(MODULE),paths=import \
	       --go-grpc_out=. --go-grpc_opt=module=$(MODULE),paths=import \
	       $(PROTO_SRC)
	@test -f $(PROTO_OUT)/auth.pb.go
	@test -f $(PROTO_OUT)/auth_grpc.pb.go
	@test -f $(PROTO_OUT)/data.pb.go
	@test -f $(PROTO_OUT)/health.pb.go
	@test -f $(PROTO_OUT)/health_grpc.pb.go
	@test -f $(PROTO_OUT)/shared.pb.go
	@test -f $(PROTO_OUT)/sync.pb.go
	@test -f $(PROTO_OUT)/sync_grpc.pb.go
	@test -f $(PROTO_OUT)/uploads.pb.go
	@test -f $(PROTO_OUT)/uploads_grpc.pb.go

proto-check:
	@echo "==> Checking proto compilation..."
	@mkdir -p $(PROTO_OUT)
	@protoc -I . \
	        --go_out=. --go_opt=module=$(MODULE),paths=import \
	        --go-grpc_out=. --go-grpc_opt=module=$(MODULE),paths=import \
	        $(PROTO_SRC)
	@test -f $(PROTO_OUT)/auth.pb.go
	@test -f $(PROTO_OUT)/auth_grpc.pb.go
	@test -f $(PROTO_OUT)/data.pb.go
	@test -f $(PROTO_OUT)/health.pb.go
	@test -f $(PROTO_OUT)/health_grpc.pb.go
	@test -f $(PROTO_OUT)/shared.pb.go
	@test -f $(PROTO_OUT)/sync.pb.go
	@test -f $(PROTO_OUT)/sync_grpc.pb.go
	@test -f $(PROTO_OUT)/uploads.pb.go
	@test -f $(PROTO_OUT)/uploads_grpc.pb.go
	@echo "Proto compilation OK"

build-server:
	go build -o bin/server ./cmd/server

build-client-cli:
	go build -o bin/client ./cmd/client/cli

build-client-tui:
	go build -o bin/client-tui ./cmd/client/tui

build-client-desktop:
	cd cmd/client/desktop/frontend && npm install && npm run build
	env CGO_LDFLAGS="-framework UniformTypeIdentifiers $$CGO_LDFLAGS" go build -tags production -o bin/client-desktop ./cmd/client/desktop

build-client-web:
	cd cmd/client/web && npm install && npm run build

dev-client-web:
	cd cmd/client/web && npm install && npm run dev

build-client: build-client-cli build-client-tui build-client-desktop build-client-web

build: build-server build-client

clean:
	rm -rf bin/ coverage.out coverage_filtered.out

dev-up:
	docker compose up -d postgres minio
	docker compose run --rm minio-init

dev-down:
	docker compose down

dev-reset:
	docker compose down -v
	rm -rf ./.db ./.minio

test-storage-integration:
	GK_TEST_MINIO_ENDPOINT=http://localhost:9000 \
	GK_TEST_MINIO_BUCKET=gophkeeper-dev \
	GK_TEST_MINIO_ACCESS_KEY=minioadmin \
	GK_TEST_MINIO_SECRET_KEY=minioadmin \
	GK_TEST_MINIO_REGION=us-east-1 \
	go test ./internal/storage -run TestS3Blob_MinIOIntegration -count=1
