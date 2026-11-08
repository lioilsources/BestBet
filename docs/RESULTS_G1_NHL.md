# Fáze 3 — brána G1: NHL

Liga `nhl`, model `periods-v1`, vygenerováno 2026-11-07 21:31 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 1,4 % lepší než základní četnosti (log-loss 1X2, 2885 zápasů); proti trhu je o 0,8 % horší (863 zápasů s kurzy).

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 1312 | 1,0375 | 1,0531 | chybí | — | 0,6256 | 0,6677 | 20,7 % / 20,2 % |
| 2025/26 (odložená) | 1312 | 1,0753 | 1,0841 | 1,0558 (n=614) | 1,0610 | 0,6509 | 0,6916 | 24,8 % / 23,4 % |
| 2026/27 (odložená) | 261 | 1,0301 | 1,0694 | 1,0132 (n=249) | 1,0301 | 0,6180 | 0,6632 | 22,6 % / 23,5 % |
| **celkem** | 2885 | 1,0540 | 1,0687 | 1,0435 (n=863) | 1,0521 | 0,6364 | 0,6782 | 22,7 % / 21,9 % |

## Sázky (01 §2.3)

Baseline, trh 1X2, sázka u betano.cz, merkurxtip.cz s posledním kurzem nejméně 1 h před začátkem, když p_model − p_implied ≥ 3 p. b. (Shin) a EV > 0. Kelly 1/4, strop 2 % bankrollu, nejvýš 5 sázek denně. CLV = log(vzatý kurz × p uzavíracího pinnacle po odmaržování).

| Sezona | Sázek | Yield | CLV | 95% interval CLV | Poráží close | Max drawdown | Bankroll na konci |
|---|---|---|---|---|---|---|---|
| 2025/26 (odložená) | 255 | -16,0 % | -4,39 % | -6,12 až -2,66 % | 10 % | 42,6 % | 0,648 |
| 2026/27 (odložená) | 110 | -13,9 % | -3,99 % | -4,95 až -3,03 % | 25 % | 12,9 % | 0,888 |
| **celkem** | 365 | -15,5 % | -4,02 % | -4,91 až -3,13 % | 23 % | 46,9 % | 0,536 |

Model vs. pohyb trhu: korelace (p_model − p_open) s (p_close − p_open) pro výhru domácích = 0,14 na 268 zápasech. Kladná korelace znamená, že model předjímá, kam se trh do začátku posune.

## Brána G1 (01 fáze 3)

| Kritérium | Práh | Hodnota | Splněno |
|---|---|---|---|
| Log-loss 1X2 vs. odmaržovaný pinnacle, odložené sezony, ≥ 3 000 zápasů | model ≤ trh − 0,5 % | model 1,0401, trh 1,0255 (+1,42 %, n=275) | ne |
| CLV sázek s hranou > 3 p. b. proti uzavíracímu kurzu | > 0, 95% interval nad nulou, ≥ 1 500 sázek | -4,02 % (-4,91 % až -3,13 %), n=120 | ne |
| Yield při Kelly 1/4 u CZ kurzů | > 0 na obou odložených sezonách | 2025/26: -16,03 % (n=255); 2026/27: -13,90 % (n=110) | ne |
| Ablace: vlna + absence zlepšují log-loss proti baseline | ve všech ligách | brankář a back-to-back zlepšují, vlna ne (`docs/RESULTS_NHL.md`) | ne |
