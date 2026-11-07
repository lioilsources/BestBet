# Fáze 3 — brána G1: HockeyAllsvenskan

Liga `swe-allsvenskan`, model `periods-v1`, vygenerováno 2026-11-07 21:31 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 6,4 % lepší než základní četnosti (log-loss 1X2, 846 zápasů); proti trhu je o 1,3 % horší (217 zápasů s kurzy).

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 364 | 0,9757 | 1,0688 | chybí | — | 0,5789 | 0,6147 | 22,8 % / 22,5 % |
| 2025/26 (odložená) | 364 | 1,0196 | 1,0646 | 1,0871 (n=99) | 1,0767 | 0,6088 | 0,6518 | 22,0 % / 23,1 % |
| 2026/27 (odložená) | 118 | 1,0060 | 1,0730 | 0,9720 (n=118) | 1,0060 | 0,5998 | 0,6411 | 23,7 % / 22,2 % |
| **celkem** | 846 | 0,9988 | 1,0676 | 1,0245 (n=217) | 1,0383 | 0,5947 | 0,6344 | 22,6 % / 22,7 % |

## Sázky (01 §2.3)

Baseline, trh 1X2, sázka u betano.cz, merkurxtip.cz s posledním kurzem nejméně 1 h před začátkem, když p_model − p_implied ≥ 3 p. b. (Shin) a EV > 0. Kelly 1/4, strop 2 % bankrollu, nejvýš 5 sázek denně. CLV = log(vzatý kurz × p uzavíracího pinnacle po odmaržování).

| Sezona | Sázek | Yield | CLV | 95% interval CLV | Poráží close | Max drawdown | Bankroll na konci |
|---|---|---|---|---|---|---|---|
| 2025/26 (odložená) | 58 | +14,5 % | -3,90 % | -12,02 až +4,21 % | 50 % | 18,8 % | 1,105 |
| 2026/27 (odložená) | 71 | -39,5 % | -7,78 % | -10,25 až -5,31 % | 18 % | 34,3 % | 0,670 |
| **celkem** | 129 | -14,4 % | -7,68 % | -10,09 až -5,27 % | 19 % | 43,4 % | 0,775 |

Model vs. pohyb trhu: korelace (p_model − p_open) s (p_close − p_open) pro výhru domácích = 0,10 na 118 zápasech. Kladná korelace znamená, že model předjímá, kam se trh do začátku posune.

## Brána G1 (01 fáze 3)

| Kritérium | Práh | Hodnota | Splněno |
|---|---|---|---|
| Log-loss 1X2 vs. odmaržovaný pinnacle, odložené sezony, ≥ 3 000 zápasů | model ≤ trh − 0,5 % | model 0,9959, trh 0,9570 (+4,07 %, n=123) | ne |
| CLV sázek s hranou > 3 p. b. proti uzavíracímu kurzu | > 0, 95% interval nad nulou, ≥ 1 500 sázek | -7,68 % (-10,09 % až -5,27 %), n=73 | ne |
| Yield při Kelly 1/4 u CZ kurzů | > 0 na obou odložených sezonách | 2025/26: +14,49 % (n=58); 2026/27: -39,48 % (n=71) | ne |
| Ablace: vlna + absence zlepšují log-loss proti baseline | ve všech ligách | žádná vrstva nezlepšila (`docs/RESULTS_PHASE2.md`) | ne |
