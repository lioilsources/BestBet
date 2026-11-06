# Rozhodnutí

Rozhodnutí, která plán nechává na implementaci (01 §6: „rozhodni sám a zapiš“). Nejnovější nahoře.

## 2026-11-06 (fáze 3, sázky a G1)

- **Sázková politika podle 01 §2.3, pevná:** jen 1X2, nejvýš jedna sázka na zápas (výběr s největší hranou), hrana = p_model − p_implied knihy po Shinovi ≥ 3 p. b. a zároveň EV > 0, kurz ≤ 10. Kelly 1/4 z bankrollu na začátku dne, strop 2 %, nejvýš 5 sázek denně. Kurz se bere poslední nejméně 1 h před začátkem.
- **CLV** = log(vzatý kurz × p_close), p_close je Shin-odmaržovaný poslední kurz reference (Pinnacle) před začátkem.
- **Knihy pro sázky:** tipsport, fortuna, chance, betano (CZ licence). Reference: pinnacle. Obojí je flag `cmd/backtest`.
- **Mock poskytovatelů (`docs/MOCK.md`)** místo klíčů, dokud je vlastník nedodá. Jen DB `*_mock`.
- **Kalibrace po trzích** (`-calib`, fit na 2024/25, hodnocení na odložených): surový model je dobře kalibrovaný; Platt ani izotonická regrese 1X2, ML ani puck line nezlepšují, u total 5,5 a remízy ve 1. třetině je zlepšení v řádu tisícin. Kalibrační vrstva se **nezapíná**; přehodnotit s NHL a s kurzy.
- **Příspěvky vstupů (07 §4):** baseline rozkládá výhru domácích na domácí led, útok a obranu obou stran přesnými Shapleyho hodnotami proti dvěma průměrným týmům na neutrálním ledě; vrstvy přidávají svůj marginální efekt.
- **NHL z veřejného API** (`api-web.nhle.com`, bez klíče): denní `score` dává góly s třetinami, OT a nájezdy (1 požadavek na den), `boxscore` brankáře. NHL je kontrolní efektivní liga (00 §1.4.5) a zdroj historie pro ≥ 3 000 zápasů brány G1.
- **Stav brány G1 na Allsvenskan (`docs/RESULTS_PHASE3.md`):** tři řádky „chybí“ (kurzy), ablace „ne“.

## 2026-11-06 (fáze 2, vlna a únava)

- **Vrstvy jako jeden zásobník** (`internal/model/layers`): každá vrstva je dvojice příznaků (x_dom, x_hosté) přičtená k log λ s koeficientem β; všechny β se fitují společně Newtonem s ridge 1, vrstva s < 100 aktivními tréninkovými zápasy se nefituje.
- **Stacking out-of-sample:** β se fitují na intenzitách baseline, které byly pro tréninkový zápas předpovězené před ním (walk-forward cache, živě z tabulky `predictions`). Fit proti baseline přefitované na stejných zápasech dával vlně záporné β: baseline zpětně „ví“ formu, kterou vlna měří. Walk proto začíná už v zahřívací sezoně a hodnotí se od `-from`.
- **Vlna = Kalmanův rating s rychlostí** (`rating.WaveTimeline`): úroveň v gólech rozdílu, rychlost na zápas, rozptyl pozorování 6, šum úrovně 0,003, šum rychlosti 0,0002, útlum rychlosti 0,9, mezi sezonami úroveň × 0,75. Pevné předem. Dvě vrstvy: `wave-level` (Kalmanova mezera minus očekávání baseline) a `wave-vel` (rozdíl rychlostí × 10).
- **Únava `b2b`:** zápas den po zápase (< 36 h) soupeře minus vlastní.
- **Výsledek na Allsvenskan (`docs/RESULTS_PHASE2.md`):** žádná vrstva log-loss nezlepšuje (goalie +0,0007, wave-level +0,0009, wave-vel +0,0003, vše +0,0024; žádná neprošla BH). Do modelu nejde nic; baseline `periods-v1` zůstává. Vlna zůstává jako UI narativ (00 §1.4.2).
- **b2b je v Allsvenskan nulová informace pro 1X2:** liga hraje celá kola pátek/sobota, takže back-to-back mají oba týmy (68 zápasů, jen 2 s jedním unaveným). Hypotéza na později: společná únava a celkový počet gólů (trh total), testovat jen na ladicí sezoně a starších.
- **Test „kopíruje vlna trh“** čeká na otevírací a uzavírací kurzy.

## 2026-11-06 (fáze 2, brankář)

- **Vrstva brankář `periods-v1+goalie`:** úprava `adj = log((1−SV_start)/(1−SV_tým))`, kde SV je stažené k ligovému průměru (prior 600 střel, poločas 365 dní) a počítané jen z dřívějších zápasů. Koeficient β se fituje v každém okně na tréninku. Konfigurace je pevná předem.
- **Výsledek na Allsvenskan (`docs/RESULTS_PHASE2.md`):** log-loss nezlepšuje (Δ = +0,0006, interval −0,0003 až +0,0015, 846 zápasů), medián β = 0,04. Vstup přitom signál má (sd adj 0,047). Podle pravidla fáze 2 vrstva **do modelu nejde**; v UI zůstává jako informace (06 §1).
- **Další hypotézy k brankáři** (testovat jen na ladicí sezoně 2024/25 a starších, nikdy na odložených): binární „chytá dvojka“ místo spojité SV%; back-to-back starty; více historie (swehockey 2019/20–2022/23) pro stabilnější SV; GSAx vyžaduje polohy střel, které Allsvenskan nemá.

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
- **Postgres na JODĚ, ne na Macu.** Na M2 neběží Docker (málo paměti). Compose v `deploy/joda`, Mac se připojuje po LAN, URL a heslo v `.env`. Stará lokální DB se nepřenášela; data se znovu nahrála z HTTP cache (`docs/DEPLOY.md`).
