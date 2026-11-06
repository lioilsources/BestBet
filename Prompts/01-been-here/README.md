# Koncept z been-here (2026-11-04 až 06)

Sada dokumentů napsaná souběžně s `../00-VLNA_PLAN.md`, původně ve větvi
`claude/emotional-betting-app-g4nzes` repa `lioilsources/been-here`.
Přenesená beze změny obsahu. **Závazný je `00-VLNA_PLAN.md` a CLAUDE.md**;
tahle sada je doplněk a kde se liší, platí 00.

## Co tu je navíc proti 00

| Soubor | Co přidává |
|---|---|
| `06_DETAIL_TYMU.md` | Obrazovka oblíbeného týmu: kádr, lavička, pozice C/LK/PK/O/B, pětky, sehrané dvojice, WOWY (spolu / odděleně / partner chybí) s prahy a smrštěním, brankářské dvojice, trenéři, stadiony, doma/venku, tour. Ke každé položce, zda smí do modelu. |
| `07_SROVNANI_TYMU.md` | Side‑by‑side srovnání v den zápasu: dvě vrstvy zvýraznění (model vs. fakt), filtry výhody (z‑rozdíl, vzorek, směr), karty „Co rozhoduje“ z dekompozice odhadu. |
| `05_APIFY_SCRAPERY.md` | Co nabízí Apify Store a MCP: Betano CZ, Pinnacle, OddsPortal (otevírací i uzavírací kurzy 20 sezon), Flashscore, Sportradar widget se sestavami a střídáním. Žádný actor na Tipsport/Fortunu. |
| `02_PROVIDERI_A_FIT.md` | Podrobnější fit matice providerů (legalita pro CZ uživatele, kurzy přes API, trhy). |
| `03_OPONENTURA.md` | Dvanáct námitek, ke každé „co by muselo platit“ a kde se to měří. |
| `04_SWOT_KONKURENCE.md` | SWOT devíti skupin konkurentů a mapa mezer. |

## Kde se liší od 00 (platí 00)

- **Stack.** Vyřešeno 2026-11-06: `01_PLAN_PRO_OPUS.md` §0 a §3 jsou přepsané
  na **Go pro všechno** (ingest, model, backtest, API, cron) v tomto repu,
  s gonum jako jedinou numerickou závislostí. Spark z 00 §hlavička není
  potřeba; data mají tisíce řádků za sezonu.
- **Ligy.** 01 začíná fotbalem top 5 + Extraliga/NHL. 00 se pivotovalo na
  hokej a okrajové ligy (HockeyAllsvenskan, Chance liga, DEL2, AHL); top ligy
  jen jako kontrolní skupina.
- **Brány.** 01 má bránu G1 (log‑loss vs. odmaržovaný Pinnacle, CLV na 1 500+
  sázkách). 00 má exit kritérium M1 (Brier/log‑loss ≤ trh a kladné CLV vs.
  Pinnacle nebo medián knih, ≥ 2 sezony). Duchem stejné; čísla bere M1.
- **Režimy.** 01 vylučuje predikční trhy a krypto úplně. 00 připouští vlastní
  trading na predikčních trzích bez propagace v appce (režim C).
- **Kombinace 1/5/10 a denní top bety** z 00 §1.4 tu nejsou; 01 má strop
  5 doporučení denně a bez pushů.

Fáze z 01 jsou v 06 §7 a 07 §7 **přemapované na milníky M0–M4 z 00 §6**
(2026-11-06): datový model do dokončení M0, WOWY a dekompozice do M1,
Soupiska/Brankáři a záložka Srovnání do M3, plný Oblíbený tým jako navržený
M3b. Zbytek textu 01–05 fáze stále používá; převodník: fáze 1–3 ≈ M0–M1,
fáze 4–5 ≈ M3, fáze 6 ≈ M3b, fáze 7 ≈ M4, fáze 8 ≈ po M4.
