# API a cron (fáze 4)

## Denní běh

| Kdy | Příkaz | Co dělá |
|---|---|---|
| 06:00 | `ingest-hockey games` | zápasy a výsledky (API-HOCKEY), rozpočet 90 požadavků |
| 06:10 | `ingest-hockey swehockey -season <aktuální>` | Allsvenskan: výsledky a brankáři |
| každých 15 min | `ingest-hockey snapshot` | kurzy pro zápasy v oknech T−24 h, T−1 h a uzavření |
| každou hodinu :20 | `predict` | přepočet živých predikcí na 72 h dopředu (idempotentní) |

Příklad pro crontab je v `deploy/crontab.example`. Na Macu běží dlouhé joby přes `nohup`.

## Endpointy (`cmd/api`, výchozí `127.0.0.1:8096`)

| Cesta | Obsah |
|---|---|
| `GET /healthz` | poslední snímek kurzů podle zdroje; `odds_stale` = žádný zdroj za 24 h |
| `GET /matches?date=YYYY-MM-DD` | zápasy dne (časové pásmo `API_TZ`), 1X2 a ML modelu, 3 nabídky s největším očekávaným log-růstem |
| `GET /matches/{id}/markets` | všechny trhy modelu, příspěvky vstupů, nabídky povolených knih s hranou, EV, Kelly |
| `GET /matches/{id}/waves` | vlna obou týmů: úroveň, směr, stabilita, 20 posledních zápasů |
| `GET /journal` | deník sázek uživatele, `Authorization: Bearer <token>` |

Pravidla, která API drží:

- Kurzy jen od knih v `API_BOOKS` (CZ licence). Ostatní knihy se nikdy nevrací, ani jako reference.
- Každá odpověď s kurzem nese upozornění podle § 5j. Kontakt na pomoc se doplní do `API_HELP`; text kontaktu musí potvrdit vlastník.
- „Zelená“ nabídka potřebuje hranu nad prahem (3 p. b. 1X2/ML, 5 p. b. ostatní), EV > 0 a trh v `API_CLV_MARKETS`. Dokud brána G1 neprošla, je seznam prázdný a nic zelené není.

## Lokálně s mockem

```sh
make mock-run
set -a && . ./.env && set +a
DATABASE_URL="$MOCK_DATABASE_URL" API_BOOKS=mock-cz1,mock-cz2 ./bin/api
curl '127.0.0.1:8096/matches?date=2026-11-07'
```
