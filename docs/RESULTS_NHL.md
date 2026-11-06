# Fáze 2 — ablace vrstev

Liga `nhl`, model `periods-v1`, vygenerováno 2026-11-06 12:31 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 1,4 % lepší než základní četnosti (log-loss 1X2, 2885 zápasů); **srovnání s trhem chybí, protože pro tyto zápasy nejsou kurzy**, takže o hraně výsledek nic neříká.

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 1312 | 1,0375 | 1,0531 | chybí | — | 0,6256 | 0,6677 | 20,7 % / 20,2 % |
| 2025/26 (odložená) | 1312 | 1,0753 | 1,0841 | chybí | — | 0,6509 | 0,6916 | 24,8 % / 23,4 % |
| 2026/27 (odložená) | 261 | 1,0301 | 1,0694 | chybí | — | 0,6180 | 0,6632 | 22,6 % / 23,5 % |
| **celkem** | 2885 | 1,0540 | 1,0687 | chybí | — | 0,6364 | 0,6782 | 22,7 % / 21,9 % |

## Ablace

Každá varianta běží na stejných oknech jako baseline; Δ je párový rozdíl log-loss 1X2 (varianta − baseline), záporné = vrstva pomáhá. Koeficienty vrstev se fitují out-of-sample: tréninkový zápas vstupuje s intenzitami baseline, které pro něj byly předpovězené před zápasem. p-hodnoty jsou korigované Benjamini–Hochberg (q = 0,10) přes jednotlivé vrstvy.

| Vrstva | Zápasů | Δ celkem | 95% interval | p | BH | Δ odložené | medián β | hodnocených s vrstvou | tréninkových s vrstvou (poslední okno) |
|---|---|---|---|---|---|---|---|---|---|
| goalie | 2885 | -0,0023 | -0,0041 až -0,0004 | 0,016 | **ano** | -0,0019 | 0,76 | 2885 | 4143 |
| wave-level | 2885 | +0,0001 | -0,0007 až +0,0010 | 0,736 | ne | -0,0003 | 0,06 | 2885 | 4138 |
| wave-vel | 2885 | -0,0002 | -0,0012 až +0,0008 | 0,691 | ne | -0,0001 | -0,55 | 2885 | 4104 |
| b2b | 2885 | -0,0025 | -0,0048 až -0,0001 | 0,038 | **ano** | -0,0011 | 0,06 | 2885 | 834 |
| vše | 2885 | -0,0040 | -0,0071 až -0,0008 | 0,015 | — | -0,0024 | 0,69 / 0,08 / -0,82 / 0,05 | 2885 | 4143 / 4138 / 4104 / 834 |

Vrstva s méně než 100 tréninkovými zápasy, kde je aktivní, se nefituje (β = 0) a nic nemění.

Vrstva zůstává jen tehdy, když Δ je záporné celkem i na odložených sezonách a projde BH. Ostatní jdou ven a zůstávají jako informace v UI.

### goalie

| Sezona | Zápasů | Δ log-loss 1X2 (goalie − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 1312 | -0,0027 | -0,0056 až +0,0003 | -0,0021 |
| 2025/26 (odložená) | 1312 | -0,0028 | -0,0054 až -0,0002 | -0,0022 |
| 2026/27 (odložená) | 261 | +0,0027 | -0,0025 až +0,0079 | +0,0013 |
| **celkem** | 2885 | -0,0023 | -0,0041 až -0,0004 | -0,0019 |

### wave-level

| Sezona | Zápasů | Δ log-loss 1X2 (wave-level − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 1312 | +0,0007 | -0,0004 až +0,0018 | +0,0005 |
| 2025/26 (odložená) | 1312 | -0,0007 | -0,0020 až +0,0006 | -0,0002 |
| 2026/27 (odložená) | 261 | +0,0015 | -0,0018 až +0,0049 | +0,0006 |
| **celkem** | 2885 | +0,0001 | -0,0007 až +0,0010 | +0,0002 |

### wave-vel

| Sezona | Zápasů | Δ log-loss 1X2 (wave-vel − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 1312 | -0,0004 | -0,0024 až +0,0017 | +0,0001 |
| 2025/26 (odložená) | 1312 | -0,0001 | -0,0010 až +0,0007 | +0,0001 |
| 2026/27 (odložená) | 261 | +0,0004 | -0,0007 až +0,0014 | +0,0004 |
| **celkem** | 2885 | -0,0002 | -0,0012 až +0,0008 | +0,0001 |

### b2b

| Sezona | Zápasů | Δ log-loss 1X2 (b2b − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 1312 | -0,0041 | -0,0078 až -0,0003 | -0,0038 |
| 2025/26 (odložená) | 1312 | -0,0006 | -0,0039 až +0,0028 | -0,0017 |
| 2026/27 (odložená) | 261 | -0,0040 | -0,0089 až +0,0008 | -0,0029 |
| **celkem** | 2885 | -0,0025 | -0,0048 až -0,0001 | -0,0027 |

### vše

| Sezona | Zápasů | Δ log-loss 1X2 (vše − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 1312 | -0,0058 | -0,0110 až -0,0005 | -0,0044 |
| 2025/26 (odložená) | 1312 | -0,0029 | -0,0073 až +0,0015 | -0,0030 |
| 2026/27 (odložená) | 261 | -0,0000 | -0,0075 až +0,0074 | -0,0011 |
| **celkem** | 2885 | -0,0040 | -0,0071 až -0,0008 | -0,0035 |

## Kopíruje vlna trh?

Korelace rezidua modelu s pohybem kurzu od otevření k uzavření (01 §5 fáze 2) vyžaduje otevírací i uzavírací kurzy. Pro Allsvenskan 2023–2026 je nemáme, takže test čeká na data.

## Kalibrace po trzích (01 fáze 3)

Kalibrace fitovaná na sezoně 2024/25 (1312 zápasů), hodnocená na odložených sezonách (1573 zápasů). Binární log-loss, nižší je lepší. Platt má 2 parametry, izotonická regrese je stupňovitá a na malém vzorku přefituje.

| Trh | Výběr | Četnost (odložené) | Průměr modelu | Log-loss surový | Platt | Izotonická |
|---|---|---|---|---|---|---|
| 1x2 | domácí | 40,2 % | 41,3 % | 0,6652 | 0,6676 | 0,6696 |
| 1x2 | remíza | 24,5 % | 23,4 % | 0,5578 | 0,5672 | **0,5576** |
| 1x2 | hosté | 35,3 % | 35,3 % | 0,6392 | 0,6403 | 0,6433 |
| ml | domácí | 52,9 % | 53,4 % | 0,6869 | 0,6876 | 0,6914 |
| total | over 5,5 | 57,5 % | 54,1 % | 0,6833 | 0,6835 | **0,6830** |
| puck_line | domácí −1,5 | 30,3 % | 27,2 % | 0,6053 | 0,6075 | 0,6101 |
| p1_1x2 | remíza ve 1. třetině | 35,0 % | 33,4 % | 0,6488 | **0,6487** | 0,6583 |
| p1_total | over 1,5 ve 1. třetině | 54,7 % | 51,9 % | 0,6873 | 0,6900 | 0,6882 |

Tučně = lepší než surový model. Spolehlivost výhry domácích (1X2) na odložených sezonách, surový model:

| Koš | Zápasů | Model | Skutečnost |
|---|---|---|---|
| 0,0 %–20,0 % | 8 | 18,5 % | 25,0 % |
| 20,0 %–40,0 % | 694 | 33,8 % | 35,9 % |
| 40,0 %–60,0 % | 834 | 46,8 % | 42,8 % |
| 60,0 %–80,0 % | 37 | 63,9 % | 67,6 % |
