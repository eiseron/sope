DEV := docker compose run --rm dev

.PHONY: tidy build test lint fmt image install

tidy:
	$(DEV) go mod tidy

build:
	$(DEV) go build -trimpath -o bin/sope ./cmd/sope

test:
	$(DEV) go test ./... -race

lint:
	$(DEV) sh -c 'test -z "$$(gofmt -l .)" || { echo "gofmt:"; gofmt -l .; exit 1; }'
	$(DEV) go vet ./...

fmt:
	$(DEV) gofmt -w .

image:
	docker build -t sope .

install:
	go install github.com/eiseron/sope/cmd/sope@latest
