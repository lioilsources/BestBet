# Rozhodnutí

Rozhodnutí, která plán nechává na implementaci (01 §6: „rozhodni sám a zapiš“). Nejnovější nahoře.

## 2026-11-08 (M1b F1: konfigurace pevně předem, zapsáno před prvním během)

- **Model `market-prior-v1`, jen 1X2.** Prior p0 = odmaržovaný (Shin) kurz v daném snímku: `opening`, `t24h`, `t1h`; `closing` jen jako diagnostika (cíl, ne sázka). Zdroj: Pinnacle, když má všechny tři výběry; jinak medián cen dostupných knih po výběrech a Shin. Úprava: s = Σ β_j·x_j, p_dom ∝ p0_dom·e^s, p_remíza ∝ p0_remíza, p_hosté ∝ p0_hosté·e^−s. β fit Newtonem na multinomické věrohodnosti, ridge 1, walk-forward po 7 dnech, trénink = dřívější zápasy se stejným snímkem a výsledkem; vrstva s < 100 aktivními tréninkovými zápasy má β = 0. Bez interceptu (trh sám = všechna β nulová).
- **Příznaky x (domácí minus hosté, kladné = pro domácí), všechny kauzální:** `goalie` = adj startéra hostů − adj startéra domácích (skutečný startér, tedy horní mez, H3); `wave-level` = Kalmanova úroveň dom − hosté (oba ≥ 5 zápasů); `wave-vel` = 10 × rozdíl rychlostí; `b2b` = b2b hostů − b2b domácích; `rest-days` = ln(1 + min(dny, 4)) dom − hosté, jen když oba mají předchozí zápas v sezoně; `pressure` = „v boji o čáru“ dom − hosté, kde tým je v boji, když odehrál ≥ 60 % základní části a |body − body čáry| ≤ zbývající zápasy × 1; čára = průměr bodů týmů na místě C a C+1 (NHL C = 16 přes celou ligu, konference se ignorují; Allsvenskan C = 10 play-off a C = 12 baráž, platí bližší čára); body NHL 2/2/1/0, Allsvenskan 3/2/1/0 (výhra, výhra po prodloužení, prohra po prodloužení, prohra); `coach-change` = 1 v prvních 10 zápasech hlavního trenéra nastoupivšího během sezony (ne v létě), dom − hosté, jen NHL (trenéři z `gamecenter/<id>/right-rail`); `periods` = ln(p_dom/p_hosté) naší baseline − totéž u p0 (přidá náš model něco k trhu?).
- **Výstupy per vrstva a snímek:** párový Δ log-loss proti trhu samotnému s 95% intervalem, p a BH (q = 0,10) přes všechny dvojice vrstva × snímek (bez `closing`); korelace (p_upr − p_open) s (p_close − p_open) proti Pinnaclu; sázky v otevíracím kurzu u betano.cz a merkurxtip.cz podle pravidla 01 §2.3 (hrana ≥ 3 p. b., EV > 0, kurz ≤ 10), CLV proti uzavíracímu Pinnaclu, flat yield.
- **H3:** podíl mezery = (LL t24h − LL t24h+goalie) / (LL t24h − LL closing) na stejných zápasech.
- **Brána A1:** vrstva projde, když (a) na snímku opening, t24h nebo t1h má Δ ≤ −0,5 % log-loss trhu a projde BH, **nebo** (b) sázky v otevíracím kurzu mají CLV > 0 s 95% intervalem nad nulou na ≥ 100 sázkách. Brankář jde do F2 jen při splnění H3 (≥ 50 % mezery a ≥ 0,5 % absolutně). Neprojde-li nic, směr A končí.
- **Okrajové ligy (H5) jsou první pohled, ne brána.**

## 2026-11-08 (po G1: směr M1b)

- **Směr po neúspěšné G1: A, plán `Prompts/02-PLAN-M1B-trh-jako-prior.md`.** Trh jako prior (00 §3.2, dosud neimplementované) a test všech předpokladů z 00 proti trhu, ne proti vlastní baseline. Dosavadní ablace vrstev měřily jen „ví vrstva něco navíc proti baseline“; hypotéza §1.4.2 „bookmaker je pomalejší než Kalman“ se měří jen proti kurzům a dosud se neměřila.
- **Rozpočet 0 Kč**, free tiery; placený tier až s kladným F1 a souhlasem.
- **Brankáři NHL jen z veřejného NHL API**, žádný scraping (DailyFaceoff vyloučen).
- **Appka (M3) zůstává no-go**, dokud neprojde brána G1b z plánu 02.
- Databáze přesunuta na ZFS `pool/bestbet/db` za běhu přes `pg_basebackup` (`docs/DEPLOY.md`). Cron na JODĚ: swehockey, nhl, predict; `games` a `snapshot` čekají na klíče.

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
- **NHL jako kontrolní liga (`docs/RESULTS_NHL.md`, 2 885 zápasů 2024/25–2026/27):** baseline je o 1,4 % lepší než základní četnosti. Na rozdíl od Allsvenskan zde pravidlem fáze 2 projdou **brankář** (Δ −0,0023, odložené −0,0019, p 0,016) a **back-to-back** (Δ −0,0025, odložené −0,0011, p 0,038); vlna (úroveň, rychlost) ne. Pro NHL platí model `periods-v1+goalie+b2b`; výběr vrstev je per liga. Kalibrace nepomáhá ani zde. B2B funguje, protože NHL hraje rozptýlený kalendář, kde unavený je obvykle jen jeden tým.
- **Vrstvy per liga v živých predikcích:** `layers` v `config/leagues.yaml`, zatím jen NHL `[goalie, b2b]`. `cmd/predict` před fitem vrstev projde historii po týdnech jako backtest, aby β stálo na out-of-sample λ (NHL ~3 s). Brankářská vrstva u nadcházejícího zápasu působí jen se známým startujícím brankářem; dokud ho ingest nemá, mění predikci jen b2b.

## 2026-11-07

- **Kurzy z OddsPapi (free tier):** historie sahá zhruba od ledna 2026, sezona 2024/25 a podzim 2025 vracejí 404. Z CZ knih jsou jen Betano CZ a MerkurXtip CZ (Tipsport, Fortuna CZ chybí). Sharp = Pinnacle. Ukládá se jen otevírací, t24h, t1h a uzavírací cena per výběr; plná historie zůstává v HTTP cache.
- **Brána G1 neprošla v žádné lize** (`docs/RESULTS_G1_SWE.md`, `docs/RESULTS_G1_NHL.md`). Allsvenskan: log-loss o 4,1 % horší než Pinnacle (n=123), CLV −7,7 % (interval −10,1 až −5,3 %), yield −14 %. NHL: o 1,4 % horší než Pinnacle (n=275), CLV −4,0 % (−4,9 až −3,1 %, n=120 se sharp close), yield −16 % a −14 %. Vrstvy (NHL −0,004 log-loss) mezeru k trhu nezavřou. Podle 00 §6 je to **no-go pro M3 (appka)**, dokud model trh neporazí; rozhodnutí o dalším směru je na vlastníkovi.
