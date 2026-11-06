# Gate before push: formatting, generated code up to date, vet, unit tests.
# Store integration tests run with `make test-db` (needs `make db`).
.PHONY: check fmt build db test-db

TEST_DATABASE_URL ?= postgres://bestbet:bestbet@127.0.0.1:5455/postgres?sslmode=disable

check:
	@out=$$(gofmt -l cmd internal); if [ -n "$$out" ]; then echo "gofmt needed: $$out"; exit 1; fi
	sqlc diff
	go vet ./...
	go test ./...

fmt:
	gofmt -w cmd internal

build:
	CGO_ENABLED=0 go build -o bin/ingest-hockey ./cmd/ingest-hockey
	CGO_ENABLED=0 go build -o bin/backtest ./cmd/backtest

db:
	docker compose up -d db

test-db:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -count=1 ./internal/store/
