DEV := docker compose run --rm dev

.PHONY: tidy build test integration lint fmt image install

tidy:
	$(DEV) go mod tidy

build:
	$(DEV) go build -trimpath -o bin/sope .

test:
	$(DEV) go test ./... -race

integration:
	$(DEV) sh -c 'BIN_DIR=/src/.cache/bin sh scripts/install-sops-age.sh && PATH=/src/.cache/bin:$$PATH go test -tags=integration ./...'

lint:
	$(DEV) sh -c 'out=$$(find . -path ./.cache -prune -o -name "*.go" -print | xargs gofmt -l); test -z "$$out" || { echo "gofmt:"; echo "$$out"; exit 1; }'
	$(DEV) go vet ./...

fmt:
	$(DEV) gofmt -w .

image:
	docker build -t sope .

install:
	go install github.com/eiseron/sope@latest
