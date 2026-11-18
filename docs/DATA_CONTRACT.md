# Datový kontrakt

Kanonické struktury jsou v `internal/canon`. Model (`internal/model`, `internal/features`, `internal/rating`) zná jen je: žádnou databázi, HTTP ani `time.Now`. Ingest převádí zdroje na tabulky (`internal/store`), `store.LoadMatches` je převádí na `canon`.

## Zápas (`canon.Match`)

| Pole | Význam |
|---|---|
| `Periods` | góly v 1.–3. třetině, nikdy prodloužení |
| `Reg` | skóre po 60 minutách = součet `Periods` |
| `Final` | konečné skóre včetně prodloužení a jednoho gólu za vítězné nájezdy |
| `DecidedIn` | `reg`, `ot`, `so` |
| `Phase` | `regular`, `pre_playoff`, `playoff`, `relegation_playoff`, `promotion_qual` |
| `Season` | rok začátku: 2025 = 2025/26 |
| `StartAt` | UTC |

`Played()` je pravda, když existují `Reg` a tři třetiny (hokej) nebo dva poločasy (fotbal). Zápas bez výsledku je jen vstup pro predikci.

### Fotbal (plán 05)

Stejné sloupce `games` a stejná pole `canon.Match`, jiný význam:

| Pole | Fotbal |
|---|---|
| `Periods` | 1. a 2. poločas (z polí „skóre před střelou“ v ASA); zápas, kde nesedí s konečným skóre, je bez poločasů |
| `Reg` (`home_60/away_60`) | skóre po základní hrací době, 90 min a nastavení |
| `ot_home/ot_away` | góly v prodloužení 2 × 15 min |
| `Final` | `Reg` + prodloužení + 1 gól vítězi penalt, pokud je znám (ASA ani football-data.co.uk ho neuvádějí) |
| `DecidedIn` | `reg`, `ot` (prodloužení), `so` (penalty) |
| `ot_type` | `none`, `et_2x15` (zápas s prodloužením), `unknown` (bez poločasů) |
| `Phase` | `regular`, `playoff` (ASA `knockout_game`) |
| `Season` | u MLS rok konce sezony: 2013–2026 kalendářní roky, 2027 přechodná, 2028 = 2027/28 |

Zavírací kurzy z football-data.co.uk nemají čas snímku; ukládají se jako `closing` minutu před výkopem (`source = fdcouk`, knihy `pinnacle`, `bet365`, `betfair_ex`, `market_max`, `market_avg`).

## Vyhodnocení trhů (hokej)

| Trh | Vyhodnocení |
|---|---|
| `1x2`, `dc`, `dnb` | 60 minut (evropská konvence, remíza existuje) |
| `ml` | včetně prodloužení a nájezdů |
| `total`, `puck_line`, `correct_score`, `btts` | 60 minut |
| `p1_1x2`, `p1_total` | 1. třetina |

Pozor: americké knihy vyhodnocují totals a puck line včetně prodloužení. Při párování kurzů s modelem se musí rozlišit (ingest to zatím neumí a značí jen `market`, viz `docs/M0.md`).

Klíče výběrů: `home`, `draw`, `away`; s linií `over@5.5`, `home@-1.5` (linie z pohledu domácích); přesný výsledek `3-2`.

## Kurzy (`canon.OddsQuote`)

Append-only snímky. `Kind`: `t24h`, `t1h`, `closing`, `historical`, `adhoc`. Backtest bere jen ceny zachycené **před začátkem** zápasu, pro každou knihu poslední takovou cenu, medián přes knihy a devig Shinovou metodou.

## Predikce (`canon.Prediction`)

| Pole | Význam |
|---|---|
| `ModelVersion` | např. `periods-v1`; mění se s každou změnou modelu |
| `InputsHash` | hash vstupů (verze, čas fitu, počet tréninkových zápasů, týmy) |
| `ComputedAt` | čas, ke kterému byl model natrénovaný; vždy před začátkem zápasu |
| `Matrix` | P(domácí = h, hosté = a) po 60 minutách, 0..12 gólů |
| `P1Matrix` | totéž pro 1. třetinu |
| `OTHomeWin` | P(domácí vyhrají prodloužení nebo nájezdy \| remíza po 60) |
| `Markets` | všechny trhy odvozené z matic (`internal/model/markets`) |
| `Contributions` | příspěvky vstupů v p. b. k výhře domácích (fáze 3) |

Backtest i produkce volají stejné `Predict`. Liší se jen zdrojem vstupů. V DB jde predikce do tabulky `predictions` s `kind = backtest | live`.

## Tabulky z migrace 0002

`players`, `player_sources`, `player_match_stats`, `player_absence_games`, `coaches`, `coach_tenures`, `units`, `unit_matches`, `duos`, `duo_matches` (WOWY: `together`, `apart`, `a_absent`, `b_absent`), `predictions`, `journal`. Význam polí podle `Prompts/01-been-here/06_DETAIL_TYMU.md` §3. Plní se od dokončení M0 (sestavy ze swehockey a hokej.cz). Zatím jsou prázdné.
