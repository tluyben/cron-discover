.PHONY: build run dev test clean

build:
	go build -o cron-discover

run: build
	./cron-discover

dev:
	go run main.go

test:
	go test ./...

clean:
	rm -f cron-discover