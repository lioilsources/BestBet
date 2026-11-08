# Nasazení

## Databáze na JODĚ

Na vývojovém Macu neběží Docker ani Postgres (rozhodnutí 2026-11-06, Mac má málo paměti). Jediný Postgres projektu běží na JODĚ a Mac se k němu připojuje po LAN: ingest, backtest, predikce, integrační testy i mock.

- Compose: `deploy/joda/compose.yml`, na JODĚ v `~/deploy/BestBet`. Postgres 17, kontejner `bestbet-db`, port 5455 jen na LAN adrese JODY. Paměť je stažená (`shared_buffers` 128 MB, 30 spojení), JODA má 3,8 GB pro všechny služby.
- `.env` na JODĚ vedle compose: `POSTGRES_PASSWORD`, `DB_BIND`. Do repa nejde.
- Data patří na ZFS: dataset `pool/bestbet` (mountpoint `/home/oli/pool-bestbet`) a pro databázi `pool/bestbet/db` s `recordsize=16k`. Cesta jde do `.env` jako `PGDATA_DIR`. Bez ní compose použije `./pgdata` na systémovém disku, který má jen ~8 GB. Přesun dat bez zastavení služby: `pg_basebackup -Ft -D -` z běžícího kontejneru rozbalit do nového adresáře (vlastník uid 70, práva 700), nastavit `PGDATA_DIR` a `up -d db`.
- Databáze na jednom serveru: `bestbet` (skutečná data), `bestbet_mock` (`make mock-run`), dočasné testovací DB z `make test-db`.
- `ingest-hockey migrate` databázi z `DATABASE_URL` založí, pokud chybí. `ingest-hockey drop-mock` smaže jen databázi s koncovkou `_mock`.

Nasazení nebo změna compose:

```sh
scp deploy/joda/compose.yml joda:deploy/BestBet/compose.yml
ssh joda 'cd ~/deploy/BestBet && docker compose up -d db'
```

Každá akce se zapíše řádkem do `~/deploy/events.jsonl` na JODĚ. Zastavuje se jen služba (`stop db`), nikdy celá compose.

## Lokální `.env`

Podle `.env.example`: `DATABASE_URL`, `TEST_DATABASE_URL` a `MOCK_DATABASE_URL` míří na JODU, heslo je z `.env` na JODĚ. Makefile `.env` načte sám; pro binárky `set -a && . ./.env && set +a`.

## Obnova dat

Data jdou celá znovu postavit z HTTP cache v `data/cache` (swehockey, NHL), bez placených API:

```sh
./bin/ingest-hockey migrate
./bin/ingest-hockey swehockey -goalies=true
./bin/ingest-hockey nhl -goalies=true
```

## Binárky a cron na JODĚ

Binárky pro Linux (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0`) jdou do `~/deploy/BestBet/bin`, k nim `config/` a `data/cache` (rsync). `DATABASE_URL` v `.env` na JODĚ míří na LAN adresu JODY, protože port nesedí na loopbacku. Cron podle `deploy/crontab.example`; bez klíčů jen řádky swehockey, nhl a predict. Crontab instaluje Olin.
