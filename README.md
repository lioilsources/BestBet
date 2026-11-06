# BestBet („Vlna“)

Výzkumný projekt: model pravděpodobností hokejových zápasů (vlna týmu + brankář) a test, jestli má proti trhu edge na okrajových ligách. Appka **nepřijímá sázky**. Plán a právní rámec: [Prompts/00-VLNA_PLAN.md](Prompts/00-VLNA_PLAN.md).

Teď běží **M0, validace dat**: sběr zápasů, brankářů a kurzů do Postgresu a report pokrytí `DATA_M0.md`.

## Rychlý start

Potřeba: Go 1.26, Docker, sqlc (jen při změně SQL).

```
make db
cp .env.example .env        # doplň klíče API-HOCKEY a OddsPapi
set -a && . ./.env && set +a
go run ./cmd/ingest-hockey migrate
go run ./cmd/ingest-hockey swehockey -league swe-allsvenskan -season 2025 -goalies=false
go run ./cmd/ingest-hockey report
```

Podrobnosti o zdrojích, pravidlech slučování, cronu a známých omezeních: [docs/M0.md](docs/M0.md).

## Struktura

- `cmd/ingest-hockey`: CLI (migrate, leagues, games, swehockey, odds, snapshot, oddspapi, report)
- `internal/db`: migrace a dotazy, `dbq` generuje sqlc
- `internal/apihockey`, `internal/oddspapi`, `internal/swehockey`: zdroje
- `internal/hockey`: výsledek 60 min vs. konečný, OT/SO, fáze, série
- `internal/store`, `internal/ingest`: slučování zdrojů, zápis
- `internal/report`: `DATA_M0.md`
- `config/leagues.yaml`: ligy, sezony, formáty prodloužení, aliasy týmů

## Testy

`make check` (gofmt, sqlc diff, vet, unit testy bez sítě). `make test-db` spustí integrační test proti lokálnímu Postgresu.
