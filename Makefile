.PHONY: build test lint proto

build-server:
	go build -o bin/server ./cmd/server

build-client:
	go build -o bin/client ./cmd/client

build: build-server build-client

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/proto/*.proto

coverage:
	go tool cover -html=coverage.out

clean:
	rm -rf bin/
