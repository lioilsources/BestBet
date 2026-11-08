# M1b F1 — vlna předpokladů proti trhu: nhl

Model `market-prior-v1`, vygenerováno 2026-11-08 03:33 UTC příkazem `backtest -m1b`. Konfigurace pevně předem: `docs/DECISIONS.md` 2026-11-08. Hodnotí se jen základní část odložených sezon (od 2025/26), kde jsou kurzy.

Prior = odmaržovaný (Shin) kurz daného snímku, Pinnacle, kde má celé 1X2, jinak medián knih. Vrstva posouvá log-šance domácí/hosté o β·x; β se fituje walk-forward po týdnech na dřívějších zápasech (ridge 1, vrstva s < 100 aktivními zápasy má β = 0, první okna s < 100 tréninkovými zápasy se přeskočí). Δ je párový rozdíl log-loss 1X2 (vrstva − trh sám) na stejných zápasech, záporné = vrstva trh zlepšuje.

## Vzorek

| Snímek | Zápasů s priorem | z toho Pinnacle | medián knih | hodnocených (po zahřátí) | Log-loss trhu |
|---|---|---|---|---|---|
| opening | 792 | 194 | 598 | 660 | 1,0418 |
| t24h | 766 | 227 | 539 | 620 | 1,0422 |
| t1h | 446 | 188 | 258 | 345 | 1,0122 |
| closing | 863 | 275 | 588 | 758 | 1,0402 |

Pressure (snímek opening): v boji o čáru je 47 % týmů v hodnocených zápasech, jen jedna strana v 143 z 792 zápasů; jen ty dávají x ≠ 0.

## Vrstvy proti trhu (log-loss)

BH (q = 0,10) přes jednotlivé vrstvy × snímky opening, t24h, t1h. `closing` je diagnostika: když vrstva pomáhá i proti uzavíracímu kurzu, trh ji nezapracoval ani do začátku.

### Snímek opening

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 660 | 660 | +0,0002 | +0,02 % | -0,0021 až +0,0025 | 0,866 | ne | 0,593 |
| wave-level | 660 | 660 | -0,0003 | -0,03 % | -0,0029 až +0,0024 | 0,843 | ne | 0,044 |
| wave-vel | 660 | 657 | +0,0011 | +0,11 % | -0,0019 až +0,0042 | 0,461 | ne | 0,815 |
| b2b | 660 | 141 | +0,0003 | +0,03 % | -0,0004 až +0,0009 | 0,409 | ne | 0,000 |
| rest-days | 660 | 524 | +0,0003 | +0,03 % | -0,0019 až +0,0026 | 0,767 | ne | 0,107 |
| pressure | 660 | 129 | +0,0002 | +0,02 % | -0,0002 až +0,0007 | 0,326 | ne | 0,000 |
| coach-change | 660 | 23 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| periods | 660 | 660 | +0,0031 | +0,30 % | -0,0015 až +0,0076 | 0,183 | ne | -0,220 |
| vše | 660 | 660 | +0,0066 | +0,64 % | -0,0001 až +0,0134 | 0,055 | — | 0,484 / 0,022 / 0,678 / -0,069 / 0,216 / 0,000 / 0,000 / -0,199 |

### Snímek t24h

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 620 | 620 | +0,0004 | +0,04 % | -0,0008 až +0,0016 | 0,529 | ne | 0,281 |
| wave-level | 620 | 620 | -0,0006 | -0,06 % | -0,0050 až +0,0038 | 0,777 | ne | 0,066 |
| wave-vel | 620 | 617 | +0,0011 | +0,11 % | -0,0018 až +0,0040 | 0,457 | ne | 0,723 |
| b2b | 620 | 134 | +0,0007 | +0,06 % | -0,0007 až +0,0020 | 0,345 | ne | -0,013 |
| rest-days | 620 | 499 | +0,0012 | +0,11 % | -0,0003 až +0,0026 | 0,107 | ne | 0,036 |
| pressure | 620 | 108 | -0,0001 | -0,01 % | -0,0010 až +0,0007 | 0,778 | ne | 0,000 |
| coach-change | 620 | 21 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| periods | 620 | 620 | +0,0031 | +0,30 % | -0,0015 až +0,0077 | 0,189 | ne | -0,081 |
| vše | 620 | 620 | +0,0086 | +0,82 % | +0,0002 až +0,0170 | 0,046 | — | 0,232 / 0,053 / 0,511 / -0,148 / 0,226 / 0,000 / 0,000 / -0,125 |

### Snímek t1h

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 345 | 345 | +0,0008 | +0,08 % | -0,0009 až +0,0025 | 0,356 | ne | 0,270 |
| wave-level | 345 | 345 | -0,0002 | -0,02 % | -0,0029 až +0,0026 | 0,903 | ne | 0,028 |
| wave-vel | 345 | 342 | +0,0004 | +0,04 % | -0,0003 až +0,0011 | 0,268 | ne | 0,043 |
| b2b | 345 | 65 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| rest-days | 345 | 279 | -0,0001 | -0,01 % | -0,0018 až +0,0017 | 0,940 | ne | 0,044 |
| pressure | 345 | 55 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| coach-change | 345 | 7 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | ne | 0,000 |
| periods | 345 | 345 | +0,0016 | +0,15 % | -0,0018 až +0,0049 | 0,365 | ne | 0,183 |
| vše | 345 | 345 | +0,0024 | +0,24 % | -0,0035 až +0,0084 | 0,416 | — | 0,319 / 0,023 / 0,061 / 0,000 / 0,098 / 0,000 / 0,000 / 0,236 |

### Snímek closing

| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95% interval | p | BH | medián β |
|---|---|---|---|---|---|---|---|---|
| goalie | 758 | 758 | +0,0004 | +0,03 % | -0,0009 až +0,0016 | 0,587 | — | 0,371 |
| wave-level | 758 | 758 | +0,0007 | +0,07 % | -0,0022 až +0,0036 | 0,645 | — | 0,042 |
| wave-vel | 758 | 755 | +0,0008 | +0,08 % | -0,0012 až +0,0028 | 0,423 | — | 0,481 |
| b2b | 758 | 163 | +0,0017 | +0,17 % | -0,0001 až +0,0035 | 0,058 | — | -0,031 |
| rest-days | 758 | 608 | +0,0012 | +0,11 % | -0,0001 až +0,0024 | 0,063 | — | 0,030 |
| pressure | 758 | 148 | +0,0003 | +0,03 % | -0,0002 až +0,0008 | 0,196 | — | 0,000 |
| coach-change | 758 | 27 | +0,0000 | +0,00 % | +0,0000 až +0,0000 | 1,000 | — | 0,000 |
| periods | 758 | 758 | +0,0016 | +0,15 % | -0,0014 až +0,0045 | 0,293 | — | -0,012 |
| vše | 758 | 758 | +0,0069 | +0,66 % | +0,0009 až +0,0129 | 0,025 | — | 0,347 / 0,030 / 0,429 / -0,173 / 0,332 / 0,000 / 0,000 / -0,023 |

## Předjímá vrstva pohyb trhu? Sázky v otevíracím kurzu

Korelace (p_upr − p_open) s (p_close − p_open) pro výhru domácích, obojí pinnacle. Sázky: otevírací kurz u betano.cz, merkurxtip.cz, hrana ≥ 3 p. b. proti jejich Shinovi, EV > 0, kurz ≤ 10, jedna sázka na zápas; CLV = log(kurz × p uzavíracího pinnacle), yield při flat vkladu.

| Varianta | Korelace s pohybem | n | Sázek | s uzavíracím Pinnaclem | CLV | 95% interval CLV | Yield |
|---|---|---|---|---|---|---|---|
| trh sám | — | 176 | 55 | 55 | +2,77 % | +0,92 až +4,62 % | -11,5 % |
| goalie | 0,154 | 176 | 90 | 84 | +1,44 % | -0,20 až +3,08 % | -12,7 % |
| wave-level | 0,264 | 176 | 99 | 71 | +1,68 % | +0,16 až +3,20 % | -3,1 % |
| wave-vel | -0,075 | 176 | 124 | 70 | +1,87 % | +0,33 až +3,41 % | -11,5 % |
| b2b | -0,032 | 176 | 59 | 59 | +2,62 % | +0,81 až +4,42 % | -17,0 % |
| rest-days | 0,088 | 176 | 89 | 81 | +1,19 % | -0,44 až +2,82 % | +0,6 % |
| pressure | -0,072 | 176 | 57 | 57 | +2,62 % | +0,81 až +4,42 % | -10,0 % |
| coach-change | — | 176 | 55 | 55 | +2,77 % | +0,92 až +4,62 % | -11,5 % |
| periods | -0,227 | 176 | 139 | 54 | +1,85 % | -0,30 až +3,99 % | -15,7 % |
| vše | 0,125 | 176 | 262 | 117 | -1,08 % | -2,55 až +0,38 % | -20,3 % |

Mimo plán (průzkumné, ne brána): sázky „trh sám“ rozdělené podle zdroje otevíracího prioru. U Pinnaclu je hrana rozdíl CZ knihy proti sharp kurzu; u mediánu jde o rozdíl mezi knihami navzájem.

| Prior | Sázek | s uzavíracím Pinnaclem | CLV | 95% interval CLV | Yield |
|---|---|---|---|---|---|
| pinnacle | 47 | 47 | +2,96 % | +0,92 až +5,00 % | -24,6 % |
| median | 8 | 8 | +1,66 % | -2,83 až +6,15 % | +65,4 % |

## H3: horní mez hodnoty brankáře

Na 620 zápasech s kurzem t24h i uzavíracím: log-loss t24h 1,0422, t24h + skutečný brankář 1,0426, uzavírací 1,0394. Mezera t24h → close je 0,0028 (0,27 %); brankář z ní vysvětlí -14 %. H3 (≥ 50 % mezery a ≥ 0,5 % absolutně): ne.

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
