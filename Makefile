# Makefile

.PHONY: build run test clean

build:
	go build -o bin/erddiagram ./cmd/server

run: build
	./bin/erddiagram

test:
	go test ./...

clean:
	go clean
	rm bin/erddiagram