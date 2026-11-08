# Gate before push: formatting, generated code up to date, vet, unit tests.
# Store integration tests run with `make test-db`. Postgres runs on JODA
# (deploy/joda, docs/DEPLOY.md); no Docker on the dev Mac. URLs come from .env.
.PHONY: check fmt build test-db mock-run mock-clean

-include .env
need = $(if $($(1)),,$(error $(1) is not set, see .env.example))

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
	CGO_ENABLED=0 go build -o bin/predict ./cmd/predict
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test-db:
	$(call need,TEST_DATABASE_URL)
	@TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -count=1 ./internal/store/

# End-to-end run without API keys: provmock stands in for API-HOCKEY and
# OddsPapi, data goes to the separate database bestbet_mock (never the real one).
MOCK_FROM ?= $(shell date -v-60d +%Y-%m-%d 2>/dev/null || date -d '-60 days' +%Y-%m-%d)
MOCK_TO ?= $(shell date +%Y-%m-%d)

mock-run: build
	$(call need,MOCK_DATABASE_URL)
	@DATABASE_URL='$(MOCK_DATABASE_URL)' ./bin/ingest-hockey migrate
	@DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey leagues -league swe-allsvenskan
	@DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey games -league swe-allsvenskan
	@DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey oddspapi -league swe-allsvenskan -from $(MOCK_FROM) -to $(MOCK_TO) -books mock-sharp,mock-cz1,mock-cz2
	@DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey odds -league swe-allsvenskan
	@DATABASE_URL='$(MOCK_DATABASE_URL)' MOCK_PROVIDERS=1 ./bin/ingest-hockey snapshot
	@DATABASE_URL='$(MOCK_DATABASE_URL)' ./bin/predict
	@DATABASE_URL='$(MOCK_DATABASE_URL)' ./bin/ingest-hockey report -out data/DATA_MOCK.md

# Drops the mock database and the mock HTTP caches (drop-mock refuses non-*_mock names).
mock-clean: build
	$(call need,MOCK_DATABASE_URL)
	@DATABASE_URL='$(MOCK_DATABASE_URL)' ./bin/ingest-hockey drop-mock
	rm -rf data/cache/apihockey-mock data/cache/oddspapi-mock
