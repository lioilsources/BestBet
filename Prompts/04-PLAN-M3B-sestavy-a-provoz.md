# Plán 04: sestavy, absence a provoz hint appky (zadání pro Opus)

Stav 2026-11-12. Navazuje na plán 03 (§7 je specifikace, tady je pořadí práce, pevná čísla a brány). Appka běží na JODĚ (`api`, port 8096), PR #1–#3 jsou v `main`.

## 1. Rozhodnutí vlastníka (platí, neptat se znovu)

- Appka je hint pro sázejícího: nesází, nepřijímá peníze, žádný affiliate. Rozpočet 0 Kč.
- Sestavy a absence jen z veřejného NHL API. Žádný scraping zpravodajství o zraněních.
- Pravidlo `market-value-v1` se nemění. Cokoli s absencemi je nová verze pravidla s vlastním deníkem a jen po bráně B2.
- Merge jen na slovo vlastníka. Každá fáze = jeden PR.

## 2. Kde to stojí

- **Hotovo:** UI Dnes / Zápas / Evaluace / Data, deník `hints`, živé kurzy přes `oddspapi -live`, předběžné ceny jako `adhoc`.
- **Čeká na vlastníka:** `ODDSPAPI_KEY` v `.env` na JODĚ. Bez něj se kurzy neobnovují a hinty nevznikají (stav 2026-11-12: klíč tam není, cron řádek taky ne).
- **Prázdné tabulky z fáze 0:** `players`, `player_sources`, `player_match_stats`, `player_absence_games`. Žádný kód je neplní.
- **Neověřeno:** že boxscore ve stavu `PRE` obsahuje sestavu. Plán 03 §7.1 to předpokládá, zkoušen byl jen zápas ve stavu `FUT` (sestava tam není). Ověřuje L0.

## 3. Fáze

### L0: ověření zdroje (půl dne, bez PR, výsledek do DECISIONS)

1. Vzít jeden dnešní zápas NHL a od −120 min do začátku každých 10 min uložit `gamecenter/{id}/boxscore`, `landing`, `right-rail` a `play-by-play` (přes `httpcache` s TTL 0, klíč s časem).
2. Zapsat, který endpoint a kolik minut před začátkem poprvé obsahuje hráče (`playerByGameStats`, případně `rosterSpots` v play-by-play) a kdy označí startujícího brankáře.
3. **Brána B0:** sestava je k dispozici ≥ 20 min před začátkem. Když ne, L2 se zmenší na „sestava po zápase + odhad z posledního zápasu“ a potvrzená sestava z plánu vypadne. Zapsat do DECISIONS a pokračovat.

Běh přes 10 min: `nohup` + `disown`, log a PID do `data/log/`.

### L1: bruslaři z boxscore (PR)

- `nhl.ParseBoxscore` vrací i bruslaře: playerId, jméno, pozice, TOI, G, A, +/−, střely, PIM. Test na fixture v `internal/nhl/testdata`.
- `ingest-hockey nhl` plní `players` + `player_sources (nhl, playerId)` + `player_match_stats`. `line_no` zůstává NULL (boxscore lajny neuvádí; nevymýšlet).
- Zpětné doplnění 2024/25–2026/27 z HTTP cache. Nové requesty jen pro zápasy, které v cache nejsou; počet nahlásit.
- Kontrola: počet bruslařů na tým a zápas je 18 (± výjimky), pokrytí zápasů ≥ 99 %. Čísla do `DATA_M0.md` přes `internal/report`.
- `/api/data` ukáže pokrytí hráčů.

### L2: sestavy a absence (PR)

- Migrace 0006 `lineups` (game_id, team_id, player_id, confirmed_at, source; PK game_id+team_id+player_id).
- `ingest-hockey nhl-lineups`: zápasy v okně −90 min až start; endpoint podle L0. Bez zápasu v okně žádný request. Po zápase doplní `player_absence_games` (důvod `unknown`).
- `internal/features/absence.go` (čistá logika nad `canon`, bez DB):
  - **pravidelný hráč:** hrál ≥ 60 % z posledních 20 zápasů týmu před daným zápasem a má průměrné TOI nad mediánem bruslařů týmu v tom okně;
  - **klíčový:** pravidelný s TOI v horní třetině týmu;
  - **`absence_toi_share`:** součet průměrného TOI chybějících pravidelných / součet průměrného TOI všech pravidelných;
  - **návrat:** pravidelný podle okna před absencí, vynechal ≥ 3 zápasy týmu, teď je v sestavě;
  - brankáři se nepočítají (mají vlastní vrstvu).
  Tato čísla jsou pevná předem a neladí se.
- Dvě varianty vstupu: `estimate` (sestava = poslední zápas týmu) a `confirmed` (tabulka `lineups`, v backtestu skutečná sestava zápasu).
- Cron řádek do `deploy/crontab.example`: `nhl-lineups` po 10 min.

### L3: UI (PR, může jít s L2)

- `/api/games/{id}`: `lineup` {status `estimate`|`confirmed`, confirmed_at, missing[], returning[], share} pro oba týmy.
- `/api/board`: počet chybějících klíčových na tým.
- Zápas: blok Sestava. Dnes: štítek „chybí N klíčových“.
- U každého čísla z modelu absencí stojí vedle pohyb ostrého trhu (invariant: metrika modelu nikdy bez trhu).
- Snímky obrazovky do PR.

### L4: evaluace (PR, report `docs/RESULTS_M3B_ABSENCE_NHL.md`)

Zapsat konfiguraci do DECISIONS **před prvním během**. Dělení sezon stejné jako F1 (DECISIONS 2026-11-08).

- **H7:** regrese [logit p_close − logit p_t24h] (Pinnacle, Shin, domácí výhra v 1X2 normovaná bez remízy) na [share_hosté − share_domácí]. Zvlášť `estimate`, `confirmed` a jejich rozdíl (nová informace). Hlásit β, 95% interval (bootstrap po zápasech), R², n. Jen zápasy s oběma snímky.
- **F-abs:** vrstva `absence` nad `periods-v1+goalie+b2b` a nad trhem jako priorem (`marketprior`), ve variantách `estimate` a `confirmed`. Metrika Δ log-loss, bootstrap interval, BH přes všechny varianty v reportu.
- **Brána B2:** vrstva jde do modelu a smí vzniknout nová verze hintu, jen když Δ log-loss proti **trhu jako prioru** < 0 s intervalem pod nulou na odložených sezonách a projde BH. Jinak zůstává absence jen informací v UI.
- V reportu vždy vedle sebe model, trh a n. Záporný výsledek se hlásí stejně jako kladný.

### L5: Allsvenskan (bez PR, dokud není co stavět)

Ověřit na swehockey, zda je sestava vidět před zápasem. Ano → návrh do DECISIONS a L1–L3 pro Allsvenskan jako další PR. Ne → zapsat a skončit.

### L6: načasování z historie kurzů (PR, report `docs/RESULTS_M3B_TIMING.md`)

Plná historie cen z OddsPapi zůstala v HTTP cache (DECISIONS 2026-11-07), do DB šly jen 4 snímky. Dá se tedy bez nových requestů změřit, jak rychle CZ knihy následují ostrý trh. Nezávisí na L1–L5, může běžet kdykoli.

- Z cache načíst celé řady 1X2 pro Pinnacle, `betano.cz` a `merkurxtip.cz` (parser `internal/oddspapi`, nic nového do `odds_snapshots`).
- **Pohyb ostrého trhu:** změna Shin pravděpodobnosti Pinnaclu o ≥ 2 p. b. během ≤ 30 min. Práh je pevný předem.
- Hlásit per kniha a liga: medián a kvartily zpoždění, než kniha posune cenu stejným směrem; podíl pohybů, po kterých kniha do začátku zápasu nezareagovala; kolik hintů `market-value-v1` by v tom okně vzniklo a jejich CLV.
- Jen popis, žádná brána a žádná změna pravidla. Výsledek říká, jestli cron po 2 h stačí, nebo okno hintu trvá kratší dobu.
- Do UI: u zápasu čas posledního pohybu ostrého trhu a zda CZ kniha už reagovala.

## 4. Provoz (průběžně)

- Jakmile vlastník ohlásí klíč v `.env` na JODĚ: přidat cron řádek pro `oddspapi -live` + `hints` z `deploy/crontab.example`, po prvním běhu zkontrolovat log a kartu Evaluace.
- Po každém mergnutém PR s kódem: build Linux binárek, `scp` do `bin/`, `ingest-hockey migrate`, `docker compose up -d api`, ověřit `/healthz` a jednu stránku.
- Jednou týdně (nebo při hlášení) stav deníku: počet hintů, kolik má close, CLV s intervalem. Bez komentáře o yieldu, dokud n < 300.
- Sledovat kvótu OddsPapi; při 429 prodloužit cron na 3 h a zapsat.

Deklarované akce na JODĚ pro tento plán: `scp` binárek a `compose.yml`, `ingest-hockey migrate`, `docker compose up -d api`, řádky crontabu z `deploy/crontab.example`. Nic jiného; žádné sudo, nikdy `compose down`. Každá akce do `~/deploy/events.jsonl`.

## 5. Pravidla práce

- Před fází přečíst tento plán, plán 03 §7, `docs/DECISIONS.md` a `CLAUDE.md`. Po fázi `make check`, PR, zápis do DECISIONS.
- Kód a komentáře anglicky, dokumenty a PR česky. Veřejné repo: žádné klíče, LAN adresy ani domovské cesty.
- Testy bez sítě a bez `time.Now`; `features` neznají DB.
- Odložené sezony se k ladění nepoužívají. Prahy z L2 se po pohledu na výsledky nemění.
- Po bráně se nezastavovat, hlásit krátce: co prošlo, čísla, co dál. Stát jen u rozhodnutí vlastníka, nevratné věci a chybějícího vstupu.

## 6. Kdy se ptát vlastníka

- B0 neprojde a zároveň existuje jiný bezplatný oficiální zdroj sestav: navrhnout, nerozhodovat.
- B2 projde: návrh nové verze hintu, než se zapíše první záznam.
- Feedback k appce (co přidat, co přehodnotit) má přednost před pořadím fází.

## 7. Zadání do chatu

> Implementuj podle `Prompts/04-PLAN-M3B-sestavy-a-provoz.md`. Začni L0. Rozhodnutí v §1 platí, neptej se na ně znovu. Hlas po každé bráně.
