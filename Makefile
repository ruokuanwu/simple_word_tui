BINARY := simpleword
PKG := ./cmd/simpleword

.PHONY: run build install test clean

run:
	go run $(PKG)

build:
	go build -o bin/$(BINARY) $(PKG)

install:
	go install $(PKG)

test:
	go test ./...

clean:
	rm -rf bin
