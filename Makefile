.PHONY: build run test test-verbose cover cover-html lint clean

BINARY := advisor
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/advisor/

run: build
	$(BUILD_DIR)/$(BINARY)

test:
	go test ./... -count=1

test-verbose:
	go test ./... -v -count=1

cover:
	go test ./... -cover -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

cover-html:
	go test ./... -cover -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html
