VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0-dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY := bin/goblin

.PHONY: build test clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/goblin

test:
	go test ./...

clean:
	rm -f $(BINARY)
