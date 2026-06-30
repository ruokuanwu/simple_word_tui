BINARY := simpleword
PKG := ./cmd/simpleword

.PHONY: run build test clean

run:
	go run $(PKG)

build:
	go build -o bin/$(BINARY) $(PKG)

test:
	go test ./...

clean:
	rm -rf bin
