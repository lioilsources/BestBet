# Gate before push: formatting, generated code up to date, vet, unit tests.
# Store integration tests run with `make test-db` (needs `make db`).
.PHONY: check fmt build db test-db mock-run

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

# End-to-end run without API keys: provmock stands in for API-HOCKEY and
# OddsPapi, data goes to the separate database bestbet_mock (never the real one).
MOCK_DATABASE_URL ?= postgres://bestbet:bestbet@127.0.0.1:5455/bestbet_mock?sslmode=disable
MOCK_FROM ?= $(shell date -v-60d +%Y-%m-%d 2>/dev/null || date -d '-60 days' +%Y-%m-%d)
MOCK_TO ?= $(shell date +%Y-%m-%d)

mock-run: build
	docker compose exec -T db psql -U bestbet -d postgres -tc "SELECT 1 FROM pg_database WHERE datname='bestbet_mock'" | grep -q 1 || docker compose exec -T db psql -U bestbet -d postgres -c "CREATE DATABASE bestbet_mock"
	DATABASE_URL='$(MOCK_DATABASE_URL)' ./bin/ingest-hockey migrate
	DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey leagues -league swe-allsvenskan
	DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey games -league swe-allsvenskan
	DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey oddspapi -league swe-allsvenskan -from $(MOCK_FROM) -to $(MOCK_TO) -books mock-sharp,mock-cz1,mock-cz2
	DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey odds -league swe-allsvenskan
	DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey snapshot
	DATABASE_URL='$(MOCK_DATABASE_URL)' ./bin/ingest-hockey report -out data/DATA_MOCK.md
