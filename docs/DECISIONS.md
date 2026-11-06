# Rozhodnutí

Rozhodnutí, která plán nechává na implementaci (01 §6: „rozhodni sám a zapiš“). Nejnovější nahoře.

## 2026-11-06

- **Sport a liga pro fáze 1–2: hokej, HockeyAllsvenskan.** Platí 00 (hokej, okrajové ligy) před 01 (fotbal top 5). Allsvenskan je jediná liga s daty v DB (4 sezony ze swehockey, brankáři u 96 % zápasů).
- **Baseline = `periods-v1`:** Poisson na góly v základní hrací době s útokem/obranou týmu, domácí výhodou ligy, časovým vážením a ridge. Fotbalový Dixon–Coles ρ se v hokeji nahrazuje parametrem navýšení remíz δ (P ∝ Poisson·(1+δ·[h=a])), protože hra bez brankáře mění konce těsných zápasů. OT/SO je logistický krok na rozdílu síly.
- **Hyperparametry pevně předem, neladí se na odložených sezonách:** poločas vážení 180 dní, ridge 5, max 12 gólů, δ zapnuté. Ladit je smí jen starší sezony (01 §6); pokud bude potřeba, přidáme ze swehockey 2019/20–2022/23.
- **Odložené sezony:** 2025/26 a 2026/27. Hodnocená 2024/25 slouží jako ladicí. Sezona 2023/24 je jen zahřívací trénink.
- **Hodnotí se jen základní část.** Play-off má vlastní multiplikátory a simulátor série (00 §1.4.7); míchat ho do baseline by zkreslilo obojí.
- **Walk-forward po 7 dnech**, trénink na všem před začátkem okna. Minimum 150 odehraných zápasů v tréninku.
- **Trh v backtestu:** poslední cena každé knihy před začátkem, medián přes knihy, devig Shin. Pinnacle pro Allsvenskan zatím nemáme; „medián knih“ odpovídá 00 §1.4.6 pro okrajové ligy.
- **Kalibrace (PAVA, Platt) je implementovaná, ale baseline ji nepoužívá.** Kalibrační vrstva přijde s fází 3, fitovaná na ladicí sezoně, nikdy na odložené.
- **Numerika:** gonum (L-BFGS) pro fit; vlastní kód jen pro PAVA, Shin a δ (01 §6).
