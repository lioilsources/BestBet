# M1b F1 — vlna předpokladů proti trhu: swe-allsvenskan

Model `market-prior-v1`, vygenerováno 2026-11-08 03:33 UTC příkazem `backtest -m1b`. Konfigurace pevně předem: `docs/DECISIONS.md` 2026-11-08. Hodnotí se jen základní část odložených sezon (od 2025/26), kde jsou kurzy.

Prior = odmaržovaný (Shin) kurz daného snímku, Pinnacle, kde má celé 1X2, jinak medián knih. Vrstva posouvá log-šance domácí/hosté o β·x; β se fituje walk-forward po týdnech na dřívějších zápasech (ridge 1, vrstva s < 100 aktivními zápasy má β = 0, první okna s < 100 tréninkovými zápasy se přeskočí). Δ je párový rozdíl log-loss 1X2 (vrstva − trh sám) na stejných zápasech, záporné = vrstva trh zlepšuje.

## Vzorek

| Snímek | Zápasů s priorem | z toho Pinnacle | medián knih | hodnocených (po zahřátí) | Log-loss trhu |
|---|---|---|---|---|---|
| opening | 200 | 92 | 108 | 83 | 0,9760 |
| t24h | 188 | 100 | 88 | 80 | 0,9748 |
| t1h | 125 | 68 | 57 | 18 | 0,9415 |
| closing | 217 | 123 | 94 | 111 | 0,9601 |

Pressure (snímek opening): v boji o čáru je 24 % týmů v hodnocených zápasech, jen jedna strana v 34 z 200 zápasů; jen ty dávají x ≠ 0.

## Vrstvy proti trhu (log-loss)

BH (q = 0,10) přes jednotlivé vrstvy × snímky opening, t24h, t1h. `closing` je diagnostika: když vrstva pomáhá i proti uzavíracímu kurzu, trh ji nezapracoval ani do začátku.

### Snímek opening

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 83 | 83 | +0,0005 | +0,05 % | -0,0008 až +0,0019 | 0,455 | ne | -0,111 |
| wave-level | 83 | 83 | +0,0045 | +0,47 % | -0,0057 až +0,0148 | 0,383 | ne | -0,061 |
| wave-vel | 83 | 83 | +0,0009 | +0,09 % | -0,0047 až +0,0064 | 0,764 | ne | 0,369 |
| b2b | 83 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| rest-days | 83 | 13 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| pressure | 83 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| coach-change | 83 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| periods | 83 | 83 | +0,0016 | +0,17 % | -0,0079 až +0,0112 | 0,737 | ne | -0,134 |
| vše | 83 | 83 | +0,0080 | +0,82 % | -0,0043 až +0,0203 | 0,201 | — | -0,217 / -0,049 / 0,409 / 0,000 / 0,000 / 0,000 / 0,000 / -0,125 |

### Snímek t24h

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 80 | 80 | +0,0017 | +0,18 % | -0,0033 až +0,0068 | 0,500 | ne | -0,500 |
| wave-level | 80 | 80 | +0,0036 | +0,37 % | -0,0008 až +0,0080 | 0,112 | ne | -0,015 |
| wave-vel | 80 | 80 | +0,0001 | +0,01 % | -0,0024 až +0,0026 | 0,917 | ne | 0,141 |
| b2b | 80 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| rest-days | 80 | 13 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| pressure | 80 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| coach-change | 80 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| periods | 80 | 80 | -0,0001 | -0,01 % | -0,0078 až +0,0076 | 0,981 | ne | -0,111 |
| vše | 80 | 80 | +0,0061 | +0,63 % | -0,0037 až +0,0159 | 0,223 | — | -0,561 / -0,018 / 0,153 / 0,000 / 0,000 / 0,000 / 0,000 / -0,110 |

### Snímek t1h

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 18 | 18 | +0,0015 | +0,16 % | -0,0001 až +0,0032 | 0,073 | ne | 0,172 |
| wave-level | 18 | 18 | -0,0027 | -0,29 % | -0,0173 až +0,0118 | 0,713 | ne | 0,000 |
| wave-vel | 18 | 18 | -0,0016 | -0,17 % | -0,0040 až +0,0008 | 0,182 | ne | 0,000 |
| b2b | 18 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| rest-days | 18 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| pressure | 18 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| coach-change | 18 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| periods | 18 | 18 | +0,0034 | +0,36 % | -0,0363 až +0,0430 | 0,868 | ne | -0,237 |
| vše | 18 | 18 | +0,0040 | +0,43 % | -0,0404 až +0,0484 | 0,860 | — | 0,044 / 0,000 / 0,000 / 0,000 / 0,000 / 0,000 / 0,000 / -0,217 |

### Snímek closing

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 111 | 111 | +0,0017 | +0,18 % | -0,0005 až +0,0039 | 0,134 | — | -0,207 |
| wave-level | 111 | 103 | +0,0046 | +0,48 % | -0,0054 až +0,0147 | 0,366 | — | -0,082 |
| wave-vel | 111 | 103 | +0,0021 | +0,22 % | -0,0021 až +0,0063 | 0,324 | — | 0,283 |
| b2b | 111 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | — | 0,000 |
| rest-days | 111 | 26 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | — | 0,000 |
| pressure | 111 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | — | 0,000 |
| coach-change | 111 | 0 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | — | 0,000 |
| periods | 111 | 111 | +0,0019 | +0,20 % | -0,0056 až +0,0095 | 0,614 | — | -0,078 |
| vše | 111 | 111 | +0,0112 | +1,17 % | -0,0001 až +0,0224 | 0,051 | — | -0,390 / -0,092 / 0,330 / 0,000 / 0,000 / 0,000 / 0,000 / -0,031 |

## Předjímá vrstva pohyb trhu? Sázky v otevíracím kurzu

Korelace (p_upr − p_open) s (p_close − p_open) pro výhru domácích, obojí pinnacle. Sázky: otevírací kurz u betano.cz, merkurxtip.cz, hrana ≥ 3 p. b. proti jejich Shinovi, EV > 0, kurz ≤ 10, jedna sázka na zápas; CLV = log(kurz × p uzavíracího pinnacle), yield při flat vkladu.

| Varianta | Korelace s pohybem | n | Sázek | s uzavíracím Pinnaclem | CLV | 95% interval CLV | Yield |
|---|---|---|---|---|---|---|---|
| trh sám | — | 60 | 25 | 25 | +0,08 % | -6,54 až +6,71 % | -31,3 % |
| goalie | -0,124 | 60 | 24 | 24 | +0,28 % | -6,63 až +7,20 % | -46,0 % |
| wave-level | -0,077 | 60 | 41 | 41 | -2,99 % | -8,47 až +2,49 % | -8,2 % |
| wave-vel | 0,384 | 60 | 26 | 26 | +2,10 % | -4,20 až +8,41 % | -28,2 % |
| b2b | — | 60 | 25 | 25 | +0,08 % | -6,54 až +6,71 % | -31,3 % |
| rest-days | — | 60 | 25 | 25 | +0,08 % | -6,54 až +6,71 % | -31,3 % |
| pressure | — | 60 | 25 | 25 | +0,08 % | -6,54 až +6,71 % | -31,3 % |
| coach-change | — | 60 | 25 | 25 | +0,08 % | -6,54 až +6,71 % | -31,3 % |
| periods | -0,098 | 60 | 32 | 32 | -0,18 % | -5,27 až +4,90 % | -28,7 % |
| vše | 0,060 | 60 | 43 | 43 | +0,34 % | -4,03 až +4,71 % | -25,3 % |

Mimo plán (průzkumné, ne brána): sázky „trh sám“ rozdělené podle zdroje otevíracího prioru. U Pinnaclu je hrana rozdíl CZ knihy proti sharp kurzu; u mediánu jde o rozdíl mezi knihami navzájem.

| Prior | Sázek | s uzavíracím Pinnaclem | CLV | 95% interval CLV | Yield |
|---|---|---|---|---|---|
| pinnacle | 25 | 25 | +0,08 % | -6,54 až +6,71 % | -31,3 % |
| median | 0 | 0 | — | — | +0,0 % |

## H3: horní mez hodnoty brankáře

Na 80 zápasech s kurzem t24h i uzavíracím: log-loss t24h 0,9748, t24h + skutečný brankář 0,9766, uzavírací 0,9483. Mezera t24h → close je 0,0265 (2,72 %); brankář z ní vysvětlí -7 %. H3 (≥ 50 % mezery a ≥ 0,5 % absolutně): ne.

## Brána A1

Vrstva projde, když (a) na snímku opening, t24h nebo t1h zlepší log-loss trhu aspoň o 0,5 % a projde BH, nebo (b) sázky v otevíracím kurzu mají CLV > 0 s 95% intervalem nad nulou na ≥ 100 sázkách.

| Vrstva | (a) log-loss | (b) CLV v open | Prošla |
|---|---|---|---|
| goalie | ne | ne | ne |
| wave-level | ne | ne | ne |
| wave-vel | ne | ne | ne |
| b2b | ne | ne | ne |
| rest-days | ne | ne | ne |
| pressure | ne | ne | ne |
| coach-change | ne | ne | ne |
| periods | ne | ne | ne |

**Verdikt A1: neprošla žádná vrstva.** Podle plánu 02 směr A končí.
