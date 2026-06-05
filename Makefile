DEV := docker compose run --rm dev

.PHONY: tidy build test integration lint fmt image install

tidy:
	$(DEV) go mod tidy

build:
	$(DEV) go build -trimpath -o bin/sope .

test:
	$(DEV) go test ./... -race

integration:
	$(DEV) sh -c 'sh scripts/install-sops-age.sh && go test -tags=integration ./...'

lint:
	$(DEV) sh -c 'test -z "$$(gofmt -l .)" || { echo "gofmt:"; gofmt -l .; exit 1; }'
	$(DEV) go vet ./...

fmt:
	$(DEV) gofmt -w .

image:
	docker build -t sope .

install:
	go install github.com/eiseron/sope@latest
