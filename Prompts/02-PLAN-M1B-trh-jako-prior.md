# 02 — Plán M1b: trh jako prior, vlna předpokladů proti trhu

Navazuje na `00-VLNA_PLAN.md` (závazný) a na výsledky brány G1
(`docs/RESULTS_G1_NHL.md`, `docs/RESULTS_G1_SWE.md`). Pracuje se po fázích,
každá má bránu; za bránou se nepokračuje bez výsledku v `docs/` a zápisu
v `docs/DECISIONS.md`. Žádné UI, žádná appka: M3 zůstává no-go, dokud
tenhle plán neprojde (00 §6).

## 0. Kde jsme (2026-11-08)

Hotovo:

- **M0.** Postgres 17 na JODĚ (ZFS `pool/bestbet/db`), 6 148 zápasů za 4 sezony
  Allsvenskan a 3 sezony NHL, brankáři u 96 % zápasů, 333 043 cenových bodů
  z OddsPapi (otevírací, t24h, t1h, uzavírací per kniha a výběr). Provider id
  pro všech 7 lig v `config/leagues.yaml`. Cron na JODĚ: ingest Allsvenskan
  a NHL denně, predikce hodinově.
- **M1 model.** Baseline `periods-v1` (Poisson na třetiny, útok/obrana,
  domácí led, navýšení remíz, časové vážení) a zásobník vrstev fitovaný
  out-of-sample. Ablace: v NHL projdou brankář a back-to-back (−0,4 % log-loss
  dohromady), vlna neprojde nikde. Allsvenskan nepustí žádnou vrstvu.
- **G1 neprošla.** Proti odmaržovanému Pinnaclu je model horší o 1,4 % (NHL)
  a 4,1 % (Allsvenskan); CLV sázek −4 % a −7,7 %, yield u Betano a MerkurXtip
  −14 až −16 %. Jediný kladný údaj: model slabě předjímá pohyb trhu od
  otevíracího k uzavíracímu kurzu (korelace 0,10–0,14).

Diagnóza: stavíme pravděpodobnost **od nuly z výsledků zápasů** a soutěžíme
s trhem, který má navíc sestavy, zranění, brankáře a peníze sharpů. To nemůže
vyjít a 00 §3.2 to předvídalo: „Prior = trh. Model začíná z implikované
pravděpodobnosti Pinnacle a upravuje ji o sestavu a vlnu.“ Tahle věta se
nikdy neimplementovala.

Důsledek pro dosavadní ablace: vrstvy jsme měřili **proti vlastní baseline**,
ne proti trhu. To odpovídá na otázku „ví vlna něco, co baseline s poločasem
180 dní neví“, ale ne na hypotézu z 00 §1.4.2: „bookmaker aktualizuje rating
pomaleji než Kalman“. Ta se měří jen proti kurzům, které od 2026-01 máme,
a dosud se neměřila (DECISIONS 2026-11-06: „test kopíruje vlna trh čeká na
kurzy“). Vlna, únava, brankář, tlak tabulky, změna trenéra a okrajové ligy
jsou tedy stále otevřené hypotézy; brankář je jen ta s největším a priori
efektem (00 §0 bod 5) a jediná s měřitelným časem příchodu informace.

### Inventura předpokladů z 00

| Předpoklad (00) | Stav | Co chybí |
|---|---|---|
| Brankář starter/backup (§0 b. 5, §1.4.5) | projde v NHL proti baseline (−0,23 %), proti trhu neměřeno | test na prioru trhu, čas ohlášení |
| Krátká a dlouhá vlna (§1.4.2) | proti baseline neprojde (Allsvenskan, NHL) | test proti trhu: předjímá vlna pohyb open → close? |
| Únava, b2b, dny odpočinku (§1.1) | b2b projde v NHL, v Allsvenskan nulová informace | test na prioru trhu; dny odpočinku jako spojitá veličina netestovány |
| Tlak: boj o titul, záchranu, derby (§1.1, §1.4.2) | **netestováno** | vrstva z tabulkového postavení a zbývajících kol; data jsou v DB |
| Změna trenéra (§1.1, §1.4.2) | **netestováno**, tabulky `coaches`, `coach_tenures` existují | vrstva „zápasů od změny“ |
| Drama posledního zápasu (§1.1) | **netestováno** | pozdní góly jsou v DB (třetiny, OT) jen hrubě; nízká priorita |
| Lineup delta mimo brankáře, hloubka lavičky (§3.1) | netestováno, v Allsvenskan a NHL bez dat o hráčích za přijatelnou cenu | mimo rozsah tohoto plánu |
| Okrajové ligy: Maxa, DEL2, Tipos (§1.4.1) | id známá, **data nestažena** | API-HOCKEY zápasy (100 požadavků/den), kurzy od 2026-01 |
| Tenké trhy: třetiny, totaly, puck line (§1.3) | model je umí, proti trhu měřeno jen 1X2 | G1 na dalších trzích z téže distribuce |
| Kombinace 1/5/10 (§1.4.4) | nemá smysl bez hrany na noze | až po G1b |
| Prior = trh (§3.2) | **neimplementováno** | F1 |

## 1. Rozhodnutí vlastníka (2026-11-08, rozhodnuto)

| Otázka | Rozhodnutí |
|---|---|
| Směr | **A:** tento plán. Alternativy v §7 jen pro záznam. |
| Rozpočet | **0 Kč.** Free tier OddsPapi a API-HOCKEY, veřejné NHL API. Placený tier přijde na stůl až s kladným výsledkem F1, a jen s výslovným souhlasem. |
| Zdroj ohlášených brankářů NHL | **Jen veřejné NHL API** (`api-web.nhle.com`). DailyFaceoff ani jiný scraping ne. Když NHL API startující brankáře před zápasem nenese, F2 to zjistí první a plán se zastaví u vlastníka. |
| Časový rámec | F1 tři až čtyři dny práce. F2 tři dny práce a **4–6 týdnů běhu** bez zásahu. F3 dva dny. |

## 1b. Jak začít (pro Opus, nová session)

1. Přečíst v tomto pořadí: `CLAUDE.md`, tento plán, `docs/DECISIONS.md`
   (celý), `docs/RESULTS_G1_NHL.md`, `docs/RESULTS_NHL.md`. Z 00 jen §1.4,
   §3 a §7. Z kódu `internal/model/layers/layers.go`, `internal/backtest`,
   `cmd/backtest/main.go`, `internal/odds`, `internal/devig`.
2. Větev `feat/m1b-market-prior` z `main` až po merge PR #1; dokud PR #1
   není mergnutý, větev z `feat/model-phase0-1`. Jeden PR pro F1.
3. Před prvním během backtestu zapsat do DECISIONS konfiguraci nových
   vrstev (`rest-days`, `pressure`, `coach-change`) a prahy brány A1.
   Potom se nemění.
4. DB je na JODĚ (`docs/DEPLOY.md`), `.env` na M2 má `DATABASE_URL`.
   Kurzy jsou v `odds_snapshots` (opening, t24h, t1h, closing). Plná
   historie cen je v HTTP cache `data/cache`, kdyby bylo potřeba jiné časy.
5. Dlouhé běhy přes `nohup` + `disown` s logem v `data/log/`.
6. Po F1 nehlásit „hotovo“, ale tabulku: vrstva × (Δ log-loss proti trhu,
   interval, korelace s pohybem, CLV v open) a verdikt A1.

## 2. Hypotézy (falzifikovatelné)

- **H1 Prior = trh.** Model `market-prior-v1`: logit odmaržované pravděpodobnosti
  trhu v čase t plus β·x vrstev, β fitované walk-forward. Předpověď: na
  kurzech t24h je log-loss aspoň o 0,5 % lepší než trh t24h sám, u aspoň
  jedné vrstvy nebo jejich součtu, po Benjamini–Hochbergovi.
- **H2 Trh zaostává za vlnou.** Rozdíl p_model − p_trh v otevíracím kurzu
  předjímá pohyb open → close (korelace dnes 0,10–0,14 s baseline od nuly).
  Měří se per vrstva: která vrstva pohyb předjímá a zda sázka v otevíracím
  kurzu podle ní má CLV > 0 proti close. Tohle je test §1.4.2 „bookmaker je
  pomalejší než Kalman“ a dělá se **jako první**, protože nepotřebuje nový sběr.
- **H3 Horní mez hodnoty brankáře.** Když do t24h kurzu vložíme brankáře,
  který opravdu chytal (hindsight), kolik z mezery t24h → close to vysvětlí.
  Méně než polovina a méně než 0,5 % absolutně → okno brankáře nemá cenu.
- **H4 Okno existuje.** Prospektivně: sázka v okamžiku ohlášení brankáře
  s modelem z H1 má CLV > 0 proti uzavíracímu Pinnaclu, interval nad nulou.
- **H5 Okrajové ligy.** Na Maxa lize, DEL2 a Tipos extralize je marže vyšší
  a rozptyl knih větší (00 §1.4.1); model na prioru trhu tam zlepší log-loss
  víc než v NHL. Kurzy jsou od 2026-01, takže je to první pohled, ne brána.

## 3. Fáze a brány

### F1 Vlna předpokladů proti trhu (data už máme, 3–4 dny)

1. `internal/model/marketprior`: vstup odmaržovaná pravděpodobnost knihy
   (Shin, Pinnacle; bez Pinnacle medián knih), výstup 1X2 a ML. Stávající
   vrstvy `layers` beze změny; β fit na logitu, ridge 1, MinN 100.
2. Nové levné vrstvy do `layers`, konfigurace pevně předem, zapsat do
   DECISIONS **před** prvním během: `rest-days` (log dnů odpočinku, rozdíl),
   `pressure` (body na záchranu a na play-off vztažené k zbývajícím kolům,
   jen poslední třetina základní části), `coach-change` (1 prvních 10 zápasů
   nového trenéra). Drama posledního zápasu a hloubka lavičky se vynechávají
   (nízká priorita, bez dat).
3. `cmd/backtest -model market-prior -prior open|t24h|t1h`: walk-forward
   po týdnech, hodnocení jen na odložených sezonách, NHL jako hlavní vzorek
   (~1 000 zápasů s kurzy), Allsvenskan jako druhý (217).
4. Výstup per vrstva a prior: log-loss proti trhu samotnému, bootstrap
   interval, BH; korelace s pohybem open → close; CLV sázky v open podle
   vrstvy (H2). Zvlášť řádek brankář hindsight (H3).
5. Souběžně stáhnout zápasy Maxa, DEL2 a Tipos přes API-HOCKEY (100
   požadavků/den, přes noc s cache) a kurzy OddsPapi od 2026-01; stejný
   report na nich jako první pohled (H5).
6. `docs/RESULTS_M1B_F1.md`, zápis do DECISIONS.

**Brána A1:** aspoň jedna vrstva zlepší prior trhu o ≥ 0,5 % po BH **nebo**
má CLV > 0 v otevíracím kurzu s intervalem nad nulou. Pak F2 jde dál jen
s vrstvami, které prošly; brankář jde do F2 jen když projde H3. Když neprojde
nic, směr A končí a plán se archivuje s výsledky; to je nejpravděpodobnější
výsledek a je to po 3–4 dnech.

### F2 Prospektivní sběr (JODA, cron, jen pro vrstvy z A1)

1. Tabulka `goalie_announcements(game_id, side, player_id, source, seen_at)`;
   zapisuje se **každá** změna, nikdy přepis. Zdroj podle §1.
2. `ingest-hockey starters`: každých 15 minut pro zápasy v oknu T−12 h až start.
3. `ingest-hockey snapshot` rozšířit o OddsPapi: Pinnacle + Betano + MerkurXtip
   pro tytéž zápasy ve stejném rytmu. Rozpočet: NHL ≈ 13 zápasů/den,
   48 volání na zápas, tedy ≈ 650 volání/den; `MinGap` 6 s to stihne.
   **Opus nejdřív ověří denní kvótu free tieru**; když nestačí, rytmus 30 minut
   a jen Pinnacle + Betano.
4. Allsvenskan jen pokud swehockey zveřejňuje sestavu před zápasem s časem;
   jinak NHL samotná.
5. Monitorování: řádek do `~/deploy/events.jsonl` při každém selhání,
   denní souhrn do `data/log/`. Běží bez zásahu 4–6 týdnů.

**Brána A2:** ≥ 300 zápasů NHL s časem ohlášení obou brankářů a s kurzem
před i po ohlášení. Méně než 150 po 6 týdnech → sběr je neudržitelný, konec.

### F3 Vyhodnocení (G1b)

1. Pro každý zápas: kurz v posledním snímku před ohlášením, kurz 30 a 60 min
   po něm, uzavírací kurz.
2. Nejdřív bez modelu: **jak rychle a kolik se trh po ohlášení pohne.**
   Pokud se pohne před ohlášením (trh brankáře zná dřív než náš zdroj),
   okno neexistuje a dál se nejde.
3. Sázky modelu z F1 v okamžiku ohlášení: CLV proti close, yield u Betano a
   MerkurXtip, 95 % intervaly, ≥ 300 sázek.
4. `docs/RESULTS_M1B_F3.md`, DECISIONS.

**Brána G1b:** CLV > 0 s intervalem nad nulou. Prošla → M2 podle 00 §6
(4 týdny papírově, pak mikro vklady, denní ledger). Neprošla → archiv.

## 4. Omezení, se kterými plán počítá

- **OddsPapi free tier:** historie od ledna 2026, nejvýš 3 knihy na volání,
  429 při rychlejším rytmu než 6 s, denní kvóta nezjištěná. Z CZ knih jen
  Betano a MerkurXtip; Tipsport a Fortuna chybí, takže yield u CZ knih je
  jen přibližný.
- **API-HOCKEY free tier:** 100 požadavků denně, stačí na `games`, ne na kurzy.
- **Rychlost informace:** oficiální sestavy přicházejí ~60 min před začátkem
  a trh je zapracuje za sekundy (00 §7 ř. 3). Celý plán stojí na tom, že
  brankáře NHL se ví dřív (ranní rozbruslení, beat reporteři). F3 bod 2 to
  měří jako první a je to nejpravděpodobnější místo, kde plán skončí.
- **Limity účtů (00 §7 ř. 12):** i při úspěchu je strop desítky tisíc Kč
  ročně. Výstupem není sázecí byznys, ale ověřený signál pro B2B feed nebo
  premium notifikaci (00 §7, scénář úspěchu).
- **Jedna osoba.** Cron musí být bezúdržbový; každé selhání je v
  `events.jsonl`, ne v hlavě.
- **Statistika:** hyperparametry pevně předem, odložené sezony se neladí,
  intervaly bootstrapem, u více testů Benjamini–Hochberg (00 §1.4.6).

## 5. Co se nedělá

Žádná appka ani UI. Z nových lig jen Maxa, DEL2 a Tipos (F1 bod 5), Extraliga
ne (Sportradar, 00 §7 ř. 3). Žádné změny baseline ani jejích hyperparametrů. Žádný placený tier bez souhlasu.
Žádné scrapování Tipsportu, Fortuny ani Flashscore (00 §7 ř. 8).

## 6. Pravidla práce pro Opus

- Před každou fází přečíst tenhle plán, `docs/DECISIONS.md` a poslední
  `docs/RESULTS_*.md`. Po fázi: `make check`, PR, zápis do DECISIONS.
- Kód a komentáře anglicky, dokumenty česky. Žádné klíče, LAN adresy ani
  domovské cesty v repu (`.env.example` s placeholderem).
- Na JODĚ jen deklarované akce: `scp` binárek, `docker compose up -d db`,
  řádky crontabu. Nikdy `compose down`, žádné sudo. Každá akce do
  `~/deploy/events.jsonl`.
- Po bráně se nezastavovat, hlásit krátce. Stát jen u rozhodnutí vlastníka,
  nevratné věci a chybějícího vstupu.

## 7. Alternativy ke směru A (pro rozhodnutí vlastníka)

| Směr | Co to je | Cena | Co by muselo platit |
|---|---|---|---|
| **A** | Tento plán. | ~7 dní práce, 4–6 týdnů běhu, 0 Kč | Brankář NHL je znám dřív, než ho trh zapracuje. |
| B | Hledat hranu v tenčích trzích: třetiny, totaly, Maxa liga, DEL2. | Vyšší: OddsPapi historii pro další ligy, stejné riziko šumu (00 §7 ř. 11) | Pinnacle tam linii nemá a knihy chybují víc; dosud nic nenaznačuje, že náš model chybuje míň. |
| C | Produkt bez sázkové hrany: deník, vlny a brankáři jako vizualizace, bez tipů. | M3 Flutter ~3 týdny, bez modelu proti trhu | Retence tipovaček bez peněz v sázce je mizerná (00 §7 ř. 9); musel by být jiný důvod otevřít. |
| D | Ukončit a archivovat. | 0 | Výstup je čistý negativní výsledek s daty, i to má cenu. |

Doporučení: A. F1 je nejlevnější test všech předpokladů z 00 najednou, proti
trhu místo proti vlastní baseline, na datech, která už máme. Po 3–4 dnech je
jasné, které předpoklady přežily a zda stojí za 6 týdnů sběru.
