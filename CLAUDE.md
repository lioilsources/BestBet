# BestBet

<!-- Max 80 řádků. Co se nevejde, jde do docs/ a sem jeden odkaz. Veřejné repo: žádné LAN IP, domovské cesty, ID. -->

## Účel
Hokejový pravděpodobnostní model („Vlna“) a test edge proti trhu na okrajových ligách. Plán: `Prompts/00-VLNA_PLAN.md` (závazný), doplněk `Prompts/01-been-here/` (fáze 0–8, specifikace týmu a srovnání).

## Build a test
```
make check      # gate před pushem: gofmt, sqlc diff, vet, go test
make db         # lokální Postgres z compose.yml (127.0.0.1:5455)
make test-db    # integrační test store proti lokálnímu Postgresu
sqlc generate   # po změně internal/db/queries nebo migrací
```

## Mapa
- `cmd/ingest-hockey`: CLI, subpříkazy viz hlavička `main.go`
- `cmd/backtest`: walk-forward backtest jedné ligy → `docs/RESULTS_PHASE*.md`
- `internal/canon`: kanonické struktury mezi ingestem, modelem a API (`docs/DATA_CONTRACT.md`)
- `internal/model/periods` (baseline), `layers` (vrstvy nad baseline), `markets`, `calib`, `explain`; `internal/devig`; `internal/backtest`
- `internal/features`, `internal/rating`: vstupy modelu (vlna, brankář, absence)
- `internal/db/migrations`: golang-migrate, embed; `internal/db/dbq` je generované sqlc
- `internal/apihockey`, `internal/oddspapi`, `internal/swehockey`: klienti a parsery zdrojů
- `internal/hockey`: výsledek, OT/SO, fáze, série (čistá logika)
- `internal/teammatch`: párování názvů týmů mezi zdroji
- `internal/store`: pravidla slučování; `internal/ingest`: kroky ingestu
- `internal/report`: generuje `DATA_M0.md`
- `config/leagues.yaml`: ligy, sezony, OT formáty, swehockey ID, aliasy

## Invarianty
- `odds_snapshots` jsou append-only, žádné UPDATE. Nový snímek = nový řádek.
- API-HOCKEY je primární pro zápasy. Sekundární zdroje jen doplňují prázdná pole (`AttachSwehockey`).
- Odds feedy týmy nezakládají. Neznámý tým = nespárovaný fixture v `fixture_links`, z toho je join rate.
- `home_60/away_60` je základ pro 1X2, `home_score/away_score` (vč. OT a gólu za nájezdy) pro ML.
- API klíče nikdy do cache klíčů ani logů (`httpcache.Request.Key` je bez klíče).
- Free tier API-HOCKEY má 100 req/den: každý nový dotaz přes `httpcache` s rozpočtem.
- Testy bez sítě a bez `time.Now`. Fixtures jsou v `internal/swehockey/testdata`, syntetická liga v `periods.Simulate`.
- `internal/model`, `features`, `rating` neznají DB ani HTTP; vstup i výstup je `canon`.
- Metrika modelu se nikdy neukazuje bez metriky trhu vedle (nebo výslovného „chybí“).
- Odložené sezony (`docs/DECISIONS.md`) se k ladění nepoužívají.
- Koeficienty vrstev se fitují jen na out-of-sample predikcích baseline (`layers.Inputs.OOS`).

## Odchylky
Žádné.

## Kam dál
`docs/M0.md` (zdroje, slučování, cron), `docs/DECISIONS.md`, `docs/DATA_CONTRACT.md`, `docs/RESULTS_PHASE*.md`, `Prompts/`.
