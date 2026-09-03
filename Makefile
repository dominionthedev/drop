.PHONY: build tidy test fmt check

build:
	go build ./...

tidy:
	go mod tidy

test:
	go test ./... -race

fmt:
	gofmt -w .

check:
	./scripts/check.sh
