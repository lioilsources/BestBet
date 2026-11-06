# Fáze 1 — baseline (bez vlny, bez brankáře)

Liga `swe-allsvenskan`, model `periods-v1`, vygenerováno 2026-11-06 07:10 UTC příkazem `backtest`. Hyperparametry: poločas vážení 180 dní, ridge 5, navýšení remíz true, max 12 gólů.

**Závěr:** model je o 6,4 % lepší než základní četnosti (log-loss 1X2, 846 zápasů); **srovnání s trhem chybí, protože pro tyto zápasy nejsou kurzy**, takže o hraně výsledek nic neříká.

Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.

| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |
|---|---|---|---|---|---|---|---|---|
| 2024/25 | 364 | 0,9757 | 1,0688 | chybí | — | 0,5789 | 0,6147 | 22,8 % / 22,5 % |
| 2025/26 (odložená) | 364 | 1,0196 | 1,0646 | chybí | — | 0,6088 | 0,6518 | 22,0 % / 23,1 % |
| 2026/27 (odložená) | 118 | 1,0060 | 1,0730 | chybí | — | 0,5998 | 0,6411 | 23,7 % / 22,2 % |
| **celkem** | 846 | 0,9988 | 1,0676 | chybí | — | 0,5947 | 0,6344 | 22,6 % / 22,7 % |
