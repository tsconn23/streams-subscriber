.PHONY: build test run run_mqtt

build:
	go build -o cmd/stream-subscriber ./cmd

run:
	cd cmd && ./stream-subscriber -cfg=./res/config.json

run_mqtt:
	cd cmd && ./stream-subscriber -cfg=./res/config-mqtt.json

test:
	go test -cover ./...
	go vet ./...