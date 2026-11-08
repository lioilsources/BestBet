# Fáze 3 — kalibrace, sázky a brána G1

Liga `swe-allsvenskan`, model `periods-v1`, vygenerováno 2026-11-06 07:49 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 6,4 % lepší než základní četnosti (log-loss 1X2, 846 zápasů); **srovnání s trhem chybí, protože pro tyto zápasy nejsou kurzy**, takže o hraně výsledek nic neříká.

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 364 | 0,9757 | 1,0688 | chybí | — | 0,5789 | 0,6147 | 22,8 % / 22,5 % |
| 2025/26 (odložená) | 364 | 1,0196 | 1,0646 | chybí | — | 0,6088 | 0,6518 | 22,0 % / 23,1 % |
| 2026/27 (odložená) | 118 | 1,0060 | 1,0730 | chybí | — | 0,5998 | 0,6411 | 23,7 % / 22,2 % |
| **celkem** | 846 | 0,9988 | 1,0676 | chybí | — | 0,5947 | 0,6344 | 22,6 % / 22,7 % |

## Kalibrace po trzích (01 fáze 3)

Kalibrace fitovaná na sezoně 2024/25 (364 zápasů), hodnocená na odložených sezonách (482 zápasů). Binární log-loss, nižší je lepší. Platt má 2 parametry, izotonická regrese je stupňovitá a na malém vzorku přefituje.

| Trh | Výběr | Četnost (odložené) | Průměr modelu | Log-loss surový | Platt | Izotonická |
|---|---|---|---|---|---|---|
| 1x2 | domácí | 42,9 % | 42,3 % | 0,6469 | 0,6606 | 0,6772 |
| 1x2 | remíza | 22,4 % | 22,9 % | 0,5300 | 0,5301 | 0,5368 |
| 1x2 | hosté | 34,6 % | 34,8 % | 0,6015 | 0,6050 | 0,6501 |
| ml | domácí | 53,5 % | 53,5 % | 0,6492 | 0,6535 | 0,6662 |
| total | over 5,5 | 44,0 % | 45,6 % | 0,6842 | **0,6824** | **0,6769** |
| puck_line | domácí −1,5 | 31,1 % | 28,1 % | 0,5955 | 0,5977 | 0,6135 |
| p1_1x2 | remíza ve 1. třetině | 33,8 % | 35,8 % | 0,6397 | **0,6389** | 0,6406 |
| p1_total | over 1,5 ve 1. třetině | 46,1 % | 45,2 % | 0,6975 | 0,6977 | 0,7051 |

Tučně = lepší než surový model. Spolehlivost výhry domácích (1X2) na odložených sezonách, surový model:

| Koš | Zápasů | Model | Skutečnost |
|---|---|---|---|
| 0,0 %–20,0 % | 28 | 16,3 % | 21,4 % |
| 20,0 %–40,0 % | 192 | 30,5 % | 31,8 % |
| 40,0 %–60,0 % | 192 | 49,3 % | 49,0 % |
| 60,0 %–80,0 % | 70 | 66,1 % | 65,7 % |

## Sázky (01 §2.3)

Baseline, trh 1X2, sázka u tipsport, fortuna, chance, betano s posledním kurzem nejméně 1 h před začátkem, když p_model − p_implied ≥ 3 p. b. (Shin) a EV > 0. Kelly 1/4, strop 2 % bankrollu, nejvýš 5 sázek denně. CLV = log(vzatý kurz × p uzavíracího pinnacle po odmaržování).

| Sezona | Sázek | Yield | CLV | 95% interval CLV | Poráží close | Max drawdown | Bankroll na konci |
|---|---|---|---|---|---|---|---|
| **celkem** | 0 | — | chybí | — | — | — | — |

Žádná sázka: pro tyto zápasy nejsou kurzy knih, u kterých smíme sázet.

Model vs. pohyb trhu: chybí (nejsou otevírací a uzavírací kurzy pinnacle).

## Brána G1 (01 fáze 3)

| Kritérium | Práh | Hodnota | Splněno |
|---|---|---|---|
| Log-loss 1X2 vs. odmaržovaný pinnacle, odložené sezony, ≥ 3 000 zápasů | model ≤ trh − 0,5 % | chybí (žádné kurzy pinnacle) | — |
| CLV sázek s hranou > 3 p. b. proti uzavíracímu kurzu | > 0, 95% interval nad nulou, ≥ 1 500 sázek | chybí (žádné sázky s uzavíracím kurzem) | — |
| Yield při Kelly 1/4 u CZ kurzů | > 0 na obou odložených sezonách | chybí (žádné sázky) | — |
| Ablace: vlna + absence zlepšují log-loss proti baseline | ve všech ligách | žádná vrstva nezlepšila (`docs/RESULTS_PHASE2.md`) | ne |
