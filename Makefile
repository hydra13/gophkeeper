.PHONY: fmt lint test cover cover-check proto proto-check build build-server build-client clean

fmt:
	goimports -w .

lint:
	golangci-lint run

test:
	go test -v -race -coverprofile=coverage.out -coverpkg="$$(go list ./... | grep -v '/pbv1' | grep -v 'proto/v1' | tr '\n' ',' | sed 's/,$$//')" $$(go list ./... | grep -v '/pbv1' | grep -v 'proto/v1')

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
	rm -rf bin/ coverage.out coverage_filtered.out
