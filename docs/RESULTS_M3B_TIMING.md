# M3b L6: načasování CZ knih proti ostrému trhu

Vygenerováno 2026-11-12 23:39 CET příkazem `backtest -timing`. Zdroj: plná historie cen OddsPapi z HTTP cache (knihy pinnacle, betano.cz, merkurxtip.cz), bez nového requestu. Plán 04 L6, prahy pevně předem.

- **Pohyb ostrého trhu:** Shin pravděpodobnost Pinnaclu (domácí nebo hosté) se změní o ≥ 2 p. b. během 30 min. Další pohyb nejdřív 30 min po předchozím.
- **Reakce knihy:** její Shin pravděpodobnost téhož výběru se posune ≥ 1 p. b. stejným směrem do začátku zápasu. **Vedla:** kniha se tak posunula už v 30 min před pohybem Pinnaclu.
- **Okno hintu:** souvislý úsek, kdy pravidlo `market-value-v1` dává hint (hrana ≥ 3 p. b., EV > 0, kurz ≤ 10). CLV prvního hintu okna proti poslední ceně Pinnaclu před začátkem.
- **Šance zachytit:** cron s periodou P a náhodnou fází okno délky d uvidí s pravděpodobností min(1, d/P).

Jen popis, bez brány. Pravidlo hintu se podle toho nemění (plán 04 §1).

## nhl

Odehraných zápasů s fixture OddsPapi: 1047, s historií v cache: 959.

### Pohyby Pinnaclu a reakce knih

Pohybů: 81 (0,08 na zápas), medián 7,2 h před začátkem.

| Kniha | Pohybů s cenou knihy | Vedla | Zareagovala | Do 2 h | Zpoždění min, medián (kvartily) |
|---|---|---|---|---|---|
| betano.cz | 78 | 18 % | 71 % | 63 % | 4 (2–25) |
| merkurxtip.cz | 81 | 14 % | 80 % | 74 % | 8 (3–40) |

### Okna hintu

Oken: 436 (0,45 na zápas). Betano 408, MerkurXtip 28. Délka v min, medián (kvartily): 7 (2–32).

| Perioda cronu | Očekávaný podíl zachycených oken |
|---|---|
| 10 min | 63 % |
| 30 min | 43 % |
| 1 h | 32 % |
| 2 h | 22 % |

| Okna | n | CLV průměr | 95% interval |
|---|---|---|---|
| všechna s uzavřením | 436 | +2,14 % | +1,57 až +2,71 % |
| otevřená do 2 h po pohybu Pinnaclu | 49 | +3,89 % | +2,32 až +5,45 % |

## swe-allsvenskan

Odehraných zápasů s fixture OddsPapi: 843, s historií v cache: 277.

### Pohyby Pinnaclu a reakce knih

Pohybů: 208 (0,75 na zápas), medián 5,4 h před začátkem.

| Kniha | Pohybů s cenou knihy | Vedla | Zareagovala | Do 2 h | Zpoždění min, medián (kvartily) |
|---|---|---|---|---|---|
| betano.cz | 174 | 5 % | 42 % | 26 % | 81 (19–397) |
| merkurxtip.cz | 182 | 9 % | 71 % | 45 % | 43 (13–293) |

### Okna hintu

Oken: 169 (0,61 na zápas). Betano 123, MerkurXtip 46. Délka v min, medián (kvartily): 73 (23–297).

| Perioda cronu | Očekávaný podíl zachycených oken |
|---|---|
| 10 min | 91 % |
| 30 min | 81 % |
| 1 h | 71 % |
| 2 h | 57 % |

| Okna | n | CLV průměr | 95% interval |
|---|---|---|---|
| všechna s uzavřením | 169 | +1,88 % | +0,13 až +3,63 % |
| otevřená do 2 h po pohybu Pinnaclu | 112 | +3,17 % | +1,25 až +5,09 % |

## Co z toho plyne pro provoz

Viz `docs/DECISIONS.md` (záznam k L6). Čísla nahoře jsou jediný podklad; interpretace se do reportu nepíše, aby šel přegenerovat.
