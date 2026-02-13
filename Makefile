.PHONY: build install test lint clean

BIN := cg
PKG := ./cmd/cg

build:
	go build -o $(BIN) $(PKG)

install:
	go install $(PKG)

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BIN)
