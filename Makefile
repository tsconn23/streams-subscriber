.PHONY: build test run

build:
	go build -o cmd/stream-subscriber ./cmd

run:
	cd cmd && ./stream-subscriber -cfg=./res/config.json

test:
	go test -cover ./...
	go vet ./...