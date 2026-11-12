# M3b L4: absence hráčů proti trhu: nhl

Vygenerováno 2026-11-12 23:06 UTC příkazem `backtest -absence`. Konfigurace pevně předem: `docs/DECISIONS.md` 2026-11-13 (L4). Prahy absencí z plánu 04 L2: pravidelný hráč ≥ 60 % z posledních 20 zápasů sezony a TOI nad mediánem týmu, brankáři mimo.

- `abs-est`: podíl chybějících minut pravidelných hráčů, hosté − domácí, sestava = poslední zápas týmu.
- `abs-conf`: totéž se skutečnou sestavou zápasu (potvrzená sestava; v backtestu zpětný pohled).
- `abs-new`: abs-conf − abs-est, tedy co přinese až potvrzená sestava.

## Vzorek

Zápasů základní části se sestavou: 2892, s absencemi spočítanými aspoň u jedné strany: 2670. abs-est ≠ 0 v 2107 zápasech (průměr |x| 0,106), abs-conf ≠ 0 v 2195 (průměr |x| 0,111).

## H7: vysvětlují absence pohyb Pinnaclu mezi t24h a uzavřením?

y = logit q_close − logit q_t24h, q = p_dom / (p_dom + p_hosté), Shin celého 1X2 Pinnaclu. OLS y = a + β·x, interval bootstrap po zápasech (2 000×). Kladné β: chybí-li víc hostům, trh posune k domácím.

| Proměnná | n | β | 95% interval | R² |
|---|---|---|---|---|
| abs-est | 176 | +0,163 | +0,003 až +0,319 | 0,0225 |
| abs-conf | 176 | +0,254 | +0,118 až +0,395 | 0,0637 |
| abs-new | 176 | +0,190 | -0,008 až +0,392 | 0,0213 |

## F-abs: absence nad trhem jako priorem

Odložené sezony od 2025/26. Prior = Shin snímku (Pinnacle, jinak medián knih). Δ = párový rozdíl log-loss 1X2 proti trhu samotnému, záporné = absence trh zlepšuje. BH (q = 0,10) přes všechny varianty s absencí mimo closing.

### Snímek opening (log-loss trhu 1,0418)

| Varianta | Zápasů | Δ log-loss | Δ / LL | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|
| bez absence | 660 | – | – | – | – | – | – |
| +abs-est | 660 | +0,0014 | +0,13 % | +0,0002 až +0,0026 | 0,026 | ne | -0,087 |
| +abs-conf | 660 | +0,0018 | +0,18 % | +0,0005 až +0,0032 | 0,009 | ne | -0,076 |

### Snímek t24h (log-loss trhu 1,0422)

| Varianta | Zápasů | Δ log-loss | Δ / LL | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|
| bez absence | 620 | – | – | – | – | – | – |
| +abs-est | 620 | +0,0013 | +0,12 % | -0,0015 až +0,0040 | 0,366 | ne | 0,231 |
| +abs-conf | 620 | +0,0020 | +0,19 % | +0,0001 až +0,0038 | 0,041 | ne | 0,035 |

### Snímek t1h (log-loss trhu 1,0122)

| Varianta | Zápasů | Δ log-loss | Δ / LL | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|
| bez absence | 345 | – | – | – | – | – | – |
| +abs-est | 345 | -0,0049 | -0,49 % | -0,0152 až +0,0054 | 0,349 | ne | -1,050 |
| +abs-conf | 345 | -0,0025 | -0,25 % | -0,0121 až +0,0071 | 0,608 | ne | -0,836 |

### Snímek closing (log-loss trhu 1,0402)

| Varianta | Zápasů | Δ log-loss | Δ / LL | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|
| bez absence | 758 | – | – | – | – | – | – |
| +abs-est | 758 | +0,0012 | +0,11 % | -0,0000 až +0,0024 | 0,058 | — | -0,050 |
| +abs-conf | 758 | +0,0014 | +0,14 % | +0,0001 až +0,0028 | 0,031 | — | -0,128 |

## F-abs: absence nad modelem

Prior = out-of-sample `periods-v1`, vrstvy goalie + b2b ve všech variantách (log-loss bez absence 1,0652). Δ proti variantě bez absence.

| Varianta | Zápasů | Δ log-loss | Δ / LL | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|
| bez absence | 1431 | – | – | – | – | – | – |
| +abs-est | 1431 | +0,0012 | +0,11 % | -0,0004 až +0,0028 | 0,144 | ne | 0,109 |
| +abs-conf | 1431 | +0,0006 | +0,05 % | -0,0003 až +0,0014 | 0,195 | ne | 0,077 |

## Brána B2

Projde abs-est na opening, t24h nebo t1h, nebo abs-conf na t1h, proti trhu jako prioru: Δ < 0, 95% interval pod nulou a BH.

| Proměnná | Snímek | Δ | Interval pod nulou | BH | Prošla |
|---|---|---|---|---|---|
| abs-est | opening | +0,0014 | ne | ne | ne |
| abs-est | t24h | +0,0013 | ne | ne | ne |
| abs-est | t1h | -0,0049 | ne | ne | ne |
| abs-conf | t1h | -0,0025 | ne | ne | ne |

**Verdikt B2: neprošlo.** Absence zůstávají jen informací v UI; model ani hint se nemění.
