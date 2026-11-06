# Fáze 2 — vrstva brankář

Liga `swe-allsvenskan`, model `periods-v1+goalie`, vygenerováno 2026-11-06 05:59 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 6,4 % lepší než základní četnosti (log-loss 1X2, 846 zápasů); **srovnání s trhem chybí, protože pro tyto zápasy nejsou kurzy**, takže o hraně výsledek nic neříká.

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 364 | 0,9763 | 1,0688 | chybí | — | 0,5792 | 0,6154 | 22,8 % / 22,5 % |
| 2025/26 (odložená) | 364 | 1,0201 | 1,0646 | chybí | — | 0,6094 | 0,6520 | 22,0 % / 23,2 % |
| 2026/27 (odložená) | 118 | 1,0070 | 1,0730 | chybí | — | 0,6004 | 0,6424 | 23,7 % / 22,2 % |
| **celkem** | 846 | 0,9994 | 1,0676 | chybí | — | 0,5951 | 0,6349 | 22,6 % / 22,7 % |

## Ablace: brankář vs. baseline

**Závěr ablace:** vrstva log-loss **nezlepšuje** (interval obsahuje nulu nebo je kladný); Δ = +0,0006 na 846 zápasech. Srovnání s trhem chybí, dokud nejsou kurzy.

| Sezona | Zápasů | Δ log-loss 1X2 (brankář − baseline) | 95% interval | Δ log-loss ML |
|---|---|---|---|---|
| 2024/25 | 364 | +0,0006 | -0,0009 až +0,0021 | +0,0007 |
| 2025/26 (odložená) | 364 | +0,0005 | -0,0009 až +0,0019 | +0,0001 |
| 2026/27 (odložená) | 118 | +0,0009 | -0,0007 až +0,0025 | +0,0013 |
| **celkem** | 846 | +0,0006 | -0,0003 až +0,0015 | +0,0006 |

Fitované β (69 oken walk-forward): min -0,36, medián 0,04, max 0,52. β = 1 znamená, že rozdíl ve stažené úspěšnosti zákroků se promítá do gólů jedna ku jedné; β kolem 0 znamená, že vrstva nic nenese. Shrinkage: prior 600 střel, poločas 365 dní.

Signál na vstupu: směrodatná odchylka adj = 0,0465 (log násobek gólů soupeře), |adj| > 0,05 u 461 z 1688 startů. Malé adj znamená, že startér se od týmového průměru skoro neliší (jednička chytá většinu zápasů).

Startující brankáři známí u obou týmů: 844 z 846 hodnocených zápasů. Ve zbytku vrstva nic nemění.
