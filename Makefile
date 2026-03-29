.PHONY: fmt lint test cover proto proto-check build build-server build-client clean

fmt:
	goimports -w .

lint:
	golangci-lint run

test:
	go test -v -race -coverprofile=coverage.out ./...

cover: test
	go tool cover -html=coverage.out

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       rpc/proto/v1/*.proto

proto-check:
	@echo "==> Checking proto compilation..."
	@protoc --go_out=. --go_opt=paths=source_relative \
	        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	        rpc/proto/v1/*.proto && echo "Proto compilation OK"

build-server:
	go build -o bin/server ./cmd/server

build-client:
	go build -o bin/client ./cmd/client/cli

build: build-server build-client

clean:
	rm -rf bin/ coverage.out
