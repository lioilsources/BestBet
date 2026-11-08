# Fáze 2 — ablace vrstev

Liga `swe-allsvenskan`, model `periods-v1`, vygenerováno 2026-11-06 07:11 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 6,4 % lepší než základní četnosti (log-loss 1X2, 846 zápasů); **srovnání s trhem chybí, protože pro tyto zápasy nejsou kurzy**, takže o hraně výsledek nic neříká.

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 364 | 0,9757 | 1,0688 | chybí | — | 0,5789 | 0,6147 | 22,8 % / 22,5 % |
| 2025/26 (odložená) | 364 | 1,0196 | 1,0646 | chybí | — | 0,6088 | 0,6518 | 22,0 % / 23,1 % |
| 2026/27 (odložená) | 118 | 1,0060 | 1,0730 | chybí | — | 0,5998 | 0,6411 | 23,7 % / 22,2 % |
| **celkem** | 846 | 0,9988 | 1,0676 | chybí | — | 0,5947 | 0,6344 | 22,6 % / 22,7 % |

## Ablace

Každá varianta běží na stejných oknech jako baseline; Δ je párový rozdíl log-loss 1X2 (varianta − baseline), záporné = vrstva pomáhá. Koeficienty vrstev se fitují out-of-sample: tréninkový zápas vstupuje s intenzitami baseline, které pro něj byly předpovězené před zápasem. p-hodnoty jsou korigované Benjamini–Hochberg (q = 0,10) přes jednotlivé vrstvy.

| Vrstva | Zápasů | Δ celkem | 95% interval | p | BH | Δ odložené | medián β | hodnocených s vrstvou | tréninkových s vrstvou (poslední okno) |
|---|---|---|---|---|---|---|---|---|---|
| goalie | 846 | +0,0007 | -0,0011 až +0,0025 | 0,450 | ne | +0,0004 | 0,10 | 846 | 1134 |
| wave-level | 846 | +0,0009 | -0,0016 až +0,0033 | 0,476 | ne | +0,0011 | 0,06 | 817 | 1151 |
| wave-vel | 846 | +0,0003 | -0,0004 až +0,0009 | 0,409 | ne | +0,0008 | -0,04 | 802 | 1136 |
| b2b | 846 | +0,0000 | +0,0000 až +0,0000 | 1,000 | ne | +0,0000 | 0,00 | 0 | 2 |
| vše | 846 | +0,0024 | -0,0007 až +0,0055 | 0,130 | — | +0,0028 | 0,12 / 0,07 / -0,15 / 0,00 | 846 | 1134 / 1151 / 1136 / 2 |

Vrstva s méně než 100 tréninkovými zápasy, kde je aktivní, se nefituje (β = 0) a nic nemění.

Vrstva zůstává jen tehdy, když Δ je záporné celkem i na odložených sezonách a projde BH. Ostatní jdou ven a zůstávají jako informace v UI.

### goalie

| Sezona | Zápasů | Δ log-loss 1X2 (goalie − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 364 | +0,0010 | -0,0007 až +0,0027 | +0,0010 |
| 2025/26 (odložená) | 364 | -0,0001 | -0,0036 až +0,0035 | -0,0002 |
| 2026/27 (odložená) | 118 | +0,0020 | -0,0017 až +0,0057 | +0,0022 |
| **celkem** | 846 | +0,0007 | -0,0011 až +0,0025 | +0,0007 |

### wave-level

| Sezona | Zápasů | Δ log-loss 1X2 (wave-level − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 364 | +0,0005 | -0,0036 až +0,0047 | -0,0005 |
| 2025/26 (odložená) | 364 | +0,0009 | -0,0029 až +0,0048 | -0,0003 |
| 2026/27 (odložená) | 118 | +0,0018 | -0,0011 až +0,0047 | +0,0021 |
| **celkem** | 846 | +0,0009 | -0,0016 až +0,0033 | -0,0001 |

### wave-vel

| Sezona | Zápasů | Δ log-loss 1X2 (wave-vel − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 364 | -0,0004 | -0,0016 až +0,0008 | -0,0002 |
| 2025/26 (odložená) | 364 | +0,0005 | -0,0002 až +0,0013 | +0,0001 |
| 2026/27 (odložená) | 118 | +0,0015 | -0,0002 až +0,0031 | +0,0012 |
| **celkem** | 846 | +0,0003 | -0,0004 až +0,0009 | +0,0001 |

### b2b

| Sezona | Zápasů | Δ log-loss 1X2 (b2b − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 364 | +0,0000 | +0,0000 až +0,0000 | +0,0000 |
| 2025/26 (odložená) | 364 | +0,0000 | +0,0000 až +0,0000 | +0,0000 |
| 2026/27 (odložená) | 118 | +0,0000 | +0,0000 až +0,0000 | +0,0000 |
| **celkem** | 846 | +0,0000 | +0,0000 až +0,0000 | +0,0000 |

### vše

| Sezona | Zápasů | Δ log-loss 1X2 (vše − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 364 | +0,0018 | -0,0028 až +0,0065 | +0,0007 |
| 2025/26 (odložená) | 364 | +0,0019 | -0,0032 až +0,0071 | +0,0001 |
| 2026/27 (odložená) | 118 | +0,0055 | +0,0001 až +0,0109 | +0,0062 |
| **celkem** | 846 | +0,0024 | -0,0007 až +0,0055 | +0,0012 |

## Kopíruje vlna trh?

Korelace rezidua modelu s pohybem kurzu od otevření k uzavření (01 §5 fáze 2) vyžaduje otevírací i uzavírací kurzy. Pro Allsvenskan 2023–2026 je nemáme, takže test čeká na data.
