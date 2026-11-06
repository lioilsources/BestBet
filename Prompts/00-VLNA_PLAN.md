# VLNA_PLAN.md — handoff pro Opus (Claude Code Director)

**Projekt:** „Vlna" (pracovní název) — emocionální sázecí/predikční appka. Porovnává dvě „vlny" (momentum + stav soupisky) dvou týmů a z jednoho pravděpodobnostního modelu odvozuje všechny běžné trhy providera (1X2, přesný výsledek 0:1/1:0…, hendikep, poločas/třetina, O/U, BTTS).
**Autor zadání:** Ol1n · **Datum:** 2026-11-04 · **Stav:** koncept → M0 validace
**Stack (default Ol1n):** Go backend (sqlc, golang-migrate, Postgres) na JODA, Flutter klient, trénink i backtest v Go (gonum; změněno 2026-11-06, původně Spark), Caddy + Cloudflare Tunnel `vlna.ol1n.com`.

---

## 0. TL;DR — rozhodnutí, která z rešerše plynou

1. **Nebýt sázkovou kanceláří.** V ČR to znamená základní povolení MF podle z. 186/2016 Sb., kauce, monitorovací napojení, AML — mimo dosah OSVČ. Appka **nepřijímá sázky**. Hraje se „za body" bez ekonomické hodnoty → mimo režim hazardu (§3 odst. 1: hazard = sázka s majetkovou hodnotou + náhoda + výhra).
2. **Tři realistické režimy fungování** (lze kombinovat, pořadí podle rizika):
   - **A) Free-to-play tipovačka + affiliate** na *licencované* CZ sázkovky (Tipsport Partners revshare 30–50 %, Fortuna 10–70 %, Betano do 30 %). Jediný legální způsob, jak v ČR „propojit" appku s reálnou sázkou.
   - **B) B2B „Vlna signals API"** pro sázkovky / tradery / media (fit s olin.now/business). Prodává se model, ne sázka.
   - **C) Vlastní trading na predikčních trzích** (Polymarket/Betfair) — vlastní kapitál, bez uživatelů. Nejčistší test, zda model má edge. Nepropagovat v appce (nejsou na whitelistu MF).
3. **Jádro = jedna matice skóre, ne N modelů.** Bivariátní Poisson / Dixon-Coles s úpravou o sestavu → z matice P(h,a) se odvodí 1X2, přesný výsledek, hendikep, O/U, BTTS, HT/FT; pro hokej model per třetina + OT/SO.
4. **„Vlna" je naučená, ne předpokládaná.** Výzkum (Ötting et al. 2025, Bundesliga, data přímo od bookmakera) ukazuje, že sázkaři na momentum sázejí ~40 % víc, ale momentum v průměru výsledky ani kurzy nemění a sázet na něj prodělává. Proto: koeficienty vlny se fitují a reportují s intervalem spolehlivosti; UI vlnu ukazuje, model ji nepřeceňuje.
5. **Absence hráčů jsou největší reálný signál** — ve fotbale expected lineups + sidelined (Sportmonks), v hokeji **potvrzený gólman** (rozdíl starter/backup cca +0,45 obdrženého gólu, hlavní hybatel linie), v e-sportu **stand-in** (náhradník = masivní vliv). Tady je nejvíc hodnoty za nejmíň peněz.
6. **Exit kritérium M1 (nesmlouvavé):** model musí na out-of-sample datech mít Brier/log-loss ≤ průměr trhu *a* kladné CLV (closing line value) vs Pinnacle (top ligy) / vs medián knih (okrajové ligy) aspoň na jedné lize přes ≥ 2 sezony. Pokud ne → projekt je „zábavná vizualizace + affiliate", nikoli „model, který ví víc".
7. **Pivot (viz §1.4):** těžiště jsou **okrajové ligy + dlouhá vlna + denní žebříček betů + žebřík kombinací 1/5/10**. Top ligy slouží jen jako kontrolní skupina. „Maximální profit" definujeme jako maximální log-růst bankrollu (Kelly) se stropem daným limity sázkovek — ne jako nejvyšší výplatu tiketu.

---

## 1. Koncept

### 1.1 Teze
Každý tým vstupuje do zápasu s určitou „vlnou": forma (posledních N zápasů vážených časem), únava (dny od posledního zápasu, cestování), tlak (boj o titul/záchranu, derby), drama posledního zápasu (inkasovaný gól v nastavení), změna trenéra, **a hlavně stav soupisky**: kdo chybí, jak kvalitní je náhrada, jak hluboká je lavička. Porovnání dvou vln → kalibrovaná pravděpodobnost každého výsledku.

### 1.2 Co je „emocionální"
- Uživatel nevidí tabulku čísel, vidí **dvě vlny** (animace, barva, amplituda) a jedno **A/B kolo** (ruleta se dvěma–třemi výsečemi podle pravděpodobnosti — stejný „wheel" pattern jako StyleMap).
- Každý trh má **vysvětlení ve větě**: „Bez Kuchty padá xG Slavie o 0,3; Plzeň má 3 dny odpočinku navíc."
- Uživatel tipuje za body, sbírá sérii, soutěží s přáteli (sociální vrstva místo peněz).

### 1.3 Trhy, které model musí umět
| Trh | Fotbal | Hokej | E-sport (CS2/LoL/Dota) |
|---|---|---|---|
| Vítěz (1X2 / ML) | z matice skóre | ML + 60min 1X2 | map winner, match winner |
| Přesný výsledek (0:1, 1:0 …) | matice P(h,a) | matice po 60 min | skóre map (2:0, 2:1) |
| Hendikep / asijský | z matice | puck line ±1,5 | map handicap ±1,5 |
| Poločas / třetina | model per půle | model per třetina (gólové intenzity 1./2./3.) | first blood / pistol round (prop) |
| O/U, BTTS | z matice | O/U 5,5 | O/U map |
| HT/FT | matice půlí | 1. třetina / konec | — |

Vše = funkce jedné pravděpodobnostní distribuce → **žádné ručně psané „tipy"**.

### 1.4 PIVOT (doplněno 2026-11-04 večer): okrajové ligy · dlouhá vlna · denní top bety · kombinace 1/5/10 · maximální profit

**Nová priorita Ol1na:** nesoutěžit s trhem v top ligách, ale hledat edge tam, kde je kapitál i pozornost řidší, držet dlouhou vlnu týmu přes sezonu, každý den vyhodnotit nejlepší bety, a skládat je do kombinací 1 / 5 / 10 zápasů s cílem maximálního profitu.

#### 1.4.1 Okrajové ligy — proč a které
- **Proč:** sharp knihy (Pinnacle) a syndikáty cenují hlavně top ligy; v nižších ligách sázkovky (a) drží vyšší marži (typicky 8–15 % vs 2–5 %), (b) reagují na informace pomaleji, (c) mají méně modelů proti sobě. Tipsport má „prakticky cokoliv" v nabídce → CZ pokrytí nižších lig je dobré.
- **Varování z výzkumu:** Winkelmann, Ötting, Deutscher & Makarewicz (JSE 2023) na 14 sezonách: neefektivity se v jednotlivých sezonách objevují, ale **nejsou perzistentní napříč ligami**, a simulace ukazují, že „signifikantní" efekt v jedné sezoně vznikne i při plné efektivitě. → Každou ligu testovat ≥ 3 sezony, korekce na mnohonásobné testování (FDR), a edge brát vážně až po replikaci v další sezoně.
- **Druhé varování:** čím okrajovější liga, tím nižší **limity** a tím rychleji sázkovka vítězný účet omezí. „Maximální profit" má tvrdý strop daný limity, ne modelem.
- **Kandidátní ligy (výběr podle 4 kritérií: kurzy ≥ 5 knih vč. 1 CZ · ≥ 3 sezony historie · žádná sharp linie · limit ≥ 2 000 Kč):**
  - Fotbal: England League One/Two + National League, Německo 3. Liga, Itálie Serie C, Španělsko Primera Federación, Skandinávie 2. úrovně (Superettan, OBOS-ligaen), **Česko FNL**, Polsko I liga, Rakousko 2. Liga.
  - Hokej: **Chance liga (CZ 2. nejvyšší)**, HockeyAllsvenskan, slovenská Tipos extraliga, DEL2, AHL.
  - E-sport tier-2: ESEA/CCT CS2, regionální LoL ligy (LFL, Prime League), Dota tier-2 — stand-in efekt je tu ještě větší než v tier-1.
- **Datová realita nižších lig:** sestavy, zranění a xG často chybí nebo přicházejí až po výkopu → signál „absence hráče" je tam slabý. **Dlouhá vlna musí model unést sama.** Historie kurzů zdarma: football-data.co.uk (~22 lig vč. 2. divizí, kurzy více knih, od 2000) → backtest bez nákladů.

#### 1.4.2 Dlouhá vlna (season-long)
- Model síly týmu jako **stavový (Kalman) rating** s driftem: pomalu klesající váha, plus explicitní skoky v přestupových oknech (obrat kádru), při změně trenéra, po postupu/sestupu (regrese nováčků, „druhá sezona").
- **Vlna = rychlost a zrychlení ratingu** ve dvou oknech: krátká (3–5 zápasů) a dlouhá (10–20 zápasů). Hypotéza k otestování: v okrajových ligách bookmaker aktualizuje svůj rating pomaleji než Kalman, takže term „rychlost" nese informaci. Pokud ne → vlna je UI narativ (viz §0 bod 4).
- Další dlouhé featury: kongesce rozlosování, motivace (o nic nejde / baráž / záchrana), zimní přestávka, domácí/venkovní trend, rozhodčí (karty, penalty — pro O/U a karty trhy), počasí/stadion (umělá tráva).

#### 1.4.3 Denní evaluace „nejlepší bety dnes"
- **Cron 06:00 + refresh T-60 min** před každým výkopem: pro každý dnešní zápas spočítat p_model, najít nejlepší kurz napříč knihami (+ nejlepší CZ kurz zvlášť), edge = p·o − 1, nejistotu modelu (bootstrap přes sezony), skóre likvidity/limitu.
- **Řadit podle očekávaného log-růstu bankrollu** (Kelly zlomek × edge), ne podle surového edge — vysoký edge na kurzu 9,0 s nejistotou ±4 p.b. je horší bet než 4 % na kurzu 1,9.
- **Veřejná účetní kniha („ledger")**: každý tip má timestamp před výkopem, férový kurz, vzatý kurz, closing kurz, výsledek, P&L; rolling 30/90/365 dní, CLV, ROI, max drawdown. Tip se nedá zpětně upravit (hash do řetězu — přirozené použití AILineage). Ledger = jediný důvěryhodný marketing a zároveň B2B důkaz.
- „Dnes" obrazovka v appce = tenhle žebříček + dvě vlny u každého zápasu.

#### 1.4.4 Kombinace 1 / 5 / 10 — matematika bez iluzí
- Akumulátor: P = Π p_i, výplata = Π o_i, **EV = Π (p_i·o_i) − 1**. Edge se násobí: +3 % na noze → 5 nohou +16 %, 10 nohou +34 % EV. Ale **stejně se násobí i marže**: nohy bez edge s marží 6 % → 5 nohou −27 %, 10 nohou −46 % EV. Kombinace zesilují to, co do nich dáš.
- Pravděpodobnost výhry: při průměrném p = 0,55 má 5-kombi ~5 %, 10-kombi ~0,25 % → rozptyl obrovský, Kelly zlomek pro 10-kombi je řádově setiny procenta bankrollu. „Maximální profit" = **maximální očekávaný log-růst** (Kelly), nikoli maximální výplata tiketu.
- **Žebřík sázek (návrh):**
  - **1 zápas** — jádro: 0,25–0,5 Kelly ze single betů s nejvyšším log-růstem (cca 70–80 % denní expozice).
  - **5 zápasů** — kombinace top-5 edge betů stejného dne, vklad ≤ 1/10 single vkladu; použít i **systémové tikety** (3/5, 4/5 — Tipsport Maxikombi/Extratip, Fortuna Multikombi), které snižují rozptyl.
  - **10 zápasů** — „loterie", vklad ≤ 1/100; smysl má jen pokud sázkovka dává **kombi bonus** (navýšení výhry za počet tipů) a bonus > kumulovaná marže → ověřit aktuální podmínky Tipsport/Fortuna/Betano/Chance (kombi bonusy se mění).
- Korelace nohou: zápasy jedné ligy v jeden den jsou většinou nezávislé, ale ne vždy (počasí, společný rozhodčí u O/U); do backtestu dát test nezávislosti.
- **Backtest žebříku:** simulace 3 sezon s reálnými limity (nižší ligy u CZ knih ~1–5 tis. Kč/tiket), s modelem omezení účtu (po N měsících výhry limit klesá) a s daní z výher (CZ: osvobození do 50 000 Kč/rok, nad to 15/23 % — ověřit aktuální znění).

#### 1.4.5 ROZHODNUTO: začínáme hokejem (2026-11-04 23:26)

**Proč hokej:** méně modelů proti nám než ve fotbale, gólman = jeden hráč s měřitelným a veřejně oznamovaným dopadem (starter vs backup ≈ +0,45 obdrženého gólu v NHL; v nižších ligách rozdíl ještě větší, protože dvojky jsou slabší), třetiny dávají přirozené trhy, a CZ sázkovky (Tipsport, Fortuna, Chance) kótují české i severské nižší soutěže.

**Ligy (pořadí nasazení):**
| # | Liga | Role | Kurzy | Data o zápasech/gólmanech |
|---|---|---|---|---|
| 1 | **HockeyAllsvenskan** (SWE 2.) | hlavní okrajová liga — nejvíc zápasů v odds feedech (OddsPapi ~364 fixtures/okno), stabilní soutěž | OddsPapi, odds-api.io, Goalserve | statistik.swehockey.se (oficiální, veřejné; sestavy vč. gólmanů), API-HOCKEY (api-sports.io) |
| 2 | **Maxa liga** (CZ 2. nejvyšší; dříve „Chance liga" — **pozor: „Chance Liga" je dnes fotbalová 1. liga**, v API feedech často stále pod starým názvem) | domácí okrajová liga, CZ sázkovky ji kótují nejhustěji | Tipsport/Fortuna/Chance (affiliate feed / snapshot), Betexplorer archiv kurzů (Livesport, jen interní backtest) | ceskyhokej.cz oficiální statistiky, hokej.cz zápisy (brankář v zápise), API-HOCKEY |
| 3 | **Tipos extraliga** (SVK) + **DEL2** (GER) | rozšíření, stejná struktura | OddsPapi, Goalserve | API-HOCKEY, oficiální weby lig |
| 4 | **Extraliga CZ** a **SHL** | „poloefektivní" kontrola | všichni | Sportradar jen pokud B2B zájem; jinak API-HOCKEY |
| K | **NHL** | efektivní kontrolní skupina (Pinnacle linie) | The Odds API, OddsPapi | NHL oficiální API (zdarma), DailyFaceoff pro gólmany |

**Datové zdroje (ověřeno 11/2026):**
- **API-HOCKEY (api-sports.io):** 262 lig vč. CZ Extraliga, 2. liga, (Chance/Maxa) liga, SVK extraliga, HockeyAllsvenskan, HockeyEttan; 19 let dat, 28 bookmakerů; free tier 100 req/den → **primární zdroj pro M0**.
- **OddsPapi:** hokej sportId 15, 376 soutěží, 145 trhů (ML, puck line, period totals), free tier + historie → kurzy pro Allsvenskan/SHL/DEL2.
- **Goalserve:** placený feed, pokrývá Chance ligu, Allsvenskan, Tipos extraligu vč. kurzů — záloha, pokud OddsPapi nemá CZ/SVK.
- **Statscore TipsterAPI:** Extraliga, Chance liga, 2. liga, Tipos extraliga — kandidát na B2B data, cena na dotaz.
- **swehockey.se:** oficiální švédské statistiky, veřejně dostupné HTML (gólmani, sestavy, střely) — scrapovat šetrně, cache.
- Gólman-signál v nižších ligách: oficiální zápis (ceskyhokej.cz / swehockey) + klubové sociální sítě ~60 min před zápasem; DailyFaceoff jen NHL.

**Hokejový model (konkrétně):**
- Gólové intenzity per třetina (λ₁, λ₂, λ₃ pro každý tým; 3. třetina má vyšší rozptyl — empty net, pull goalie) → matice skóre po 60 min; OT (3v3, 5 min) + nájezdy jako samostatný binární krok (≈ 50/50 s malou korekcí na sílu).
- **Gólman:** rating z SV% / GSAx (kde jsou xG) s shrinkagem k ligovému průměru podle počtu startů; `goalie_delta = starter − expected_backup`; stav potvrzení `UNCONFIRMED → EXPECTED → CONFIRMED` s časovou značkou.
- Dlouhá vlna: Kalman rating týmu + rychlost; kongesce (back-to-back je v hokeji silný efekt), cestování, domácí led.
- Trhy: ML (vč. OT) a 1X2 (60 min) odděleně — **EU knihy kótují 60 min s remízou, US knihy vč. OT** (pozor na míchání ve feedech); puck line ±1,5; O/U 5,5; 1. třetina 1X2 a O/U 1,5; přesný výsledek 60 min; HT/FT ekvivalent = „vede po 1. třetině / vítěz".
- Kombinace: hokejové večery mají 5–8 zápasů jedné ligy ve stejný čas → přirozený materiál pro 5-kombi; korelaci přes počasí neřešit, ale testovat korelaci O/U uvnitř ligy po změnách pravidel.

**První kroky pro Directora (M0-hokej, 1 týden):**
1. Repo `lioilsources/Vlna`, `cmd/ingest-hockey` (Go), Postgres schéma: `leagues, teams, games(period_scores, ot, so), goalies(game, team, starter, sv, ga, toi), odds_snapshots(book, market, line, price, captured_at)`.
2. Stáhnout přes API-HOCKEY 3 sezony: HockeyAllsvenskan, Maxa/Chance liga, Tipos extraliga, Extraliga, NHL (results + period scores). Zjistit, zda API-HOCKEY vrací gólmany per zápas; kde ne → swehockey/ceskyhokej scraper s cache.
3. OddsPapi: historické kurzy pro Allsvenskan + SHL (ML/1X2, puck line, totals, 1. třetina); pro CZ ligy snapshot z affiliate XML feedu Tipsportu (po schválení v Tipsport Partners) nebo ruční export Betexplorer archivu pro backtest.
4. Report `DATA_M0.md`: pokrytí per liga (zápasy, % zápasů s kurzy, % s gólmanem, průměrná marže 1X2, počet knih), a doporučení, které 2 ligy jdou do M1.

#### 1.4.7 Play-off je jiná disciplína (doplněno 2026-11-04 23:43)

Play-off porušuje tři předpoklady základního modelu: nezávislost zápasů, stabilní gólové intenzity, rotaci gólmanů. Řešení:

1. **`phase` jako proměnná prvního řádu:** `regular | pre_playoff (zajištěno/o nic nejde) | playoff | relegation_playoff (baráž) | promotion_qual (kval)`. Detekce z API-HOCKEY `stage/round` + ručně udržovaný kalendář formátů per liga (formáty se mění — Maxa liga/Extraliga baráž, Allsvenskan→SHL kval: **ověřit aktuální formát každou sezonu**).
2. **Play-off nefitovat od nuly** (n je malé): regular-season parametry = prior; play-off má jen multiplikátory estimované poolem přes ligy a sezony: `λ_po = λ_reg · k_λ` (očekávání 0,85–0,95), `home_adv_po`, pravidla OT (evropská play-off typicky 20 min 5v5 do rozhodnutí, bez nájezdů → P(remíza po 60) se rozděluje podle síly s větší variancí, ne 50/50).
3. **Stav série jako featury:** číslo zápasu, vedení v sérii, eliminační zápas, dny odpočinku, pořadí domácího ledu (2-2-1-1-1). Hypotéza: trh přeceňuje poslední zápas série („momentum", „hot goalie"); G7 modelovat s vyšší variancí blízko 50/50.
4. **Top-heavy síla týmu:** v play-off roste TOI top-6/top-4/startéra o 20–30 %; proxy = výkon v těsných zápasech a proti top-8 v regular season; player-level TOI pokud zdroj dá (NHL ano, nižší ligy zřídka).
5. **Gólman:** starter chytá vše → `goalie_delta ≈ 0`, jistota potvrzení ↑; výskyt dvojky = nejsilnější signál sezony (zranění/kolaps). Zranění bruslařů se tají → spoléhat na zápisy, ne oznámení.
6. **Edge play-off = odvozené trhy série:** Monte Carlo zbytku série ze zápasových pravděpodobností + pořadí ledu → vítěz série, přesný výsledek série (4:1, 4:2…), vedení po N zápasech. V nižších ligách bookmaker tyhle trhy cení ručně a nekonzistentně; model je má z jedné distribuce zadarmo. **Priorita pro M1-playoff.**
7. **Cross-league kalibrace pro baráž/kval:** ratingy různých lig nejsou srovnatelné; kalibrovat přes historické baráže + CHL/pohárové zápasy; bez kalibrace model mlčí.
8. **Riziko a kombinace:** zápasy téže série jsou korelované → **kombi nikdy se dvěma zápasy jedné série**; 5-kombi přes víc sérií; nižší Kelly zlomek (vyšší rozptyl); v backtestu samostatný kbelík „play-off" poolovaný přes ligy, jinak FDR nic nepustí.
9. **Pre-playoff:** posledních ~5 kol regular season s flagem „zajištěno"/„bojuje" — nižší ligy to v kurzech podceňují častěji než NHL; levný test.

**Dopad na M0:** do schématu `games` přidat `phase`, `series_id`, `game_no`, `series_score_before`, `ot_type` (none/5v5_20/3v3_5/shootout); z period scores oddělit OT/SO góly.
**Dopad na M1:** report přínosu vrstev zvlášť pro regular a play-off; simulátor série (`cmd/series-sim`) jako samostatný artefakt.

#### 1.4.6 Co to mění v plánu
- M0: **hokej první** (§1.4.5) — API-HOCKEY + OddsPapi + swehockey/ceskyhokej; fotbal (football-data.co.uk) až jako druhý sport, NHL jako kontrolní „efektivní" skupina.
- M1: vyhodnocení **per liga** s FDR korekcí, min. 2 sezony OOS, flat-stake ROI na ≥ 500 single betech, CLV vs medián knih (sharp linie neexistuje); simulace žebříku 1/5/10 v log-růstu; samostatný report přínosu vrstvy „gólman".
- M2 (režim C) běží na vlastním účtu u CZ sázkovky (Tipsport/Fortuna — legální, zdaněné), ne na Polymarket.
- Produkt: „Dnes" + ledger jsou první obrazovky; v hokeji je druhá obrazovka „Brankáři" (potvrzení, delta), ne sestava.

---

## 2. Fit s providery (rešerše 11/2026)

### 2.1 Český trh
| Provider | Postavení | API / data | Affiliate | Fit pro Vlnu |
|---|---|---|---|---|
| **Tipsport** (+Chance v holdingu) | jednička, online ~94 % sázek, čistý zisk > 3,5 mld. Kč, nejširší nabídka, Tip Liga/Tipcup komunita, e-sport průkopník | žádné veřejné API; affiliate dashboard má **XML feedy (main odds, live)**; existuje Apify scraper Tipsport odds (ToS riziko) | Tipsport Partners: revshare 30–50 %, bez negative carryover, jen CZ/SK traffic, výplata měsíčně | **Primární affiliate** (největší konverze) + kandidát B2B |
| **Fortuna** | dvojka, FEG (CZ/SK/PL), vlastní liga naming | bez API; kurzy přes agregátory (OddsPapi má Fortuna PL/RO — CZ ověřit) | revshare 10–70 %, bez CPA | sekundární affiliate |
| **Betano** | nováček po pádu povinnosti registrace na pobočce; nejlepší UX, roste nejrychleji, sponzor Sparty | bez API | do 30 %, **negative carryover**, CPA/sub-aff | affiliate č. 3; zajímavý **B2B kandidát** (nový hráč hledá diferenciaci) |
| Sazkabet (Allwyn), Synot Tip, Forbet | menší | bez API | standardní | doplňkové |

Komunita kolem sázení v ČR už existuje (Tip Liga, fóra Tipsportu/Fortuny) → Vlna soupeří o pozornost, ne o licenci.

### 2.2 Globální
| Provider | Role pro Vlnu |
|---|---|
| **Pinnacle** | *benchmark pravdy*: nízká marže, sharp — closing line = reference pro CLV. Dostupný v The Odds API / OddsPapi. |
| **Betfair Exchange** | back/lay = nejbližší „A/B" trh; API pro zákazníky; zdroj implikovaných pravděpodobností bez marže |
| bet365, Unibet, William Hill | kurzy přes agregátory; affiliate mimo CZ (pro CZ uživatele nepropagovat) |

### 2.3 E-sport
- Data o zápasech/soupiskách: PandaScore, GRID, Abios (enterprise, řádově 2–10 k USD/měs.). Levně: odds-api.io (15 titulů pod jedním slugem, WS streaming) nebo OddsPapi (CS2/LoL/Dota, 150+ knih na zápas).
- Sázkovky: Thunderpick (nejhlubší e-sport krypto kniha), GG.bet, Rivalry; Tipsport má e-sport, ale úzký.
- **Fit: vysoký.** Stand-in logika je přesně „vliv absentujícího hráče"; roster changes jsou veřejné (HLTV/Liquipedia). Komunita mladá, mobilní.

### 2.4 Krypto knihy
Stake (největší handle), Cloudbet (od 2013, vysoké limity, 2026 růst basketbal/e-sport), Thunderpick, BC.Game — všechny Curaçao, **nejsou na whitelistu MF → propagace v ČR je nelegální reklama** (z. 40/1995 §2 odst. 1 písm. f; MF v 2026 výslovně varuje influencery). Použitelné jen jako datový zdroj kurzů, ne jako partner.

### 2.5 Predikční trhy („pravděpodobnostní A/B")
- **Polymarket**: po akvizici CFTC-licencované burzy spuštěn v USA (05/2026), mix objemu přešel z politiky na **sport**; CLOB API + WS.
- **Kalshi**: CFTC, ~80 % objemu sport, RSA-PSS API, jen US.
- Agregace: odds-api.io (depth + lay odds pro Polymarket/Kalshi), Prediction Hunt API, Predexon.
- **Fit:** binární YES/NO = přesně „A/B ruleta". Pro CZ uživatele právně šedé (nelicencováno) → **režim C (vlastní test edge), ne produkt pro uživatele**.

### 2.6 Doporučený provider-fit
1. **Odds feed**: OddsPapi (370 knih vč. sharps + crypto + exchange, free tier, historická data zdarma) nebo odds-api.io (250+ knih, Polymarket/Kalshi depth, Starter £99/měs.). The Odds API jako fallback (~40–80 knih, EU region vč. Pinnacle/Betfair). **Ověřit pokrytí Tipsport/Fortuna CZ** — pokud chybí, CZ kurzy jen z affiliate XML feedu nebo scraper.
2. **Fotbalová data**: Sportmonks Starter €29/měs. (lineups, sidelined, xG, predictions; add-on **Expected Lineups** = přesně náš signál). API-Football free 100 req/den pro prototyp.
3. **Hokej**: NHL oficiální API zdarma (sportly SDK), DailyFaceoff/beat-reporter pro gólmany; **Extraliga** = Sportradar Global Ice Hockey (enterprise) nebo scraping Flashscore (Livesport, Praha — ToS riziko). Začít NHL, Extraligu až po B2B zájmu.
4. **E-sport**: odds-api.io/OddsPapi pro kurzy, Liquipedia/HLTV pro rostery (scraping s respektem k robots).

---

## 3. Model „Vlna"

### 3.1 Architektura
```
[ingest] fixtures, results, lineups, sidelined, odds snapshots (T-24h, T-1h, closing)
   ↓
[features]
  team strength: Dixon-Coles attack/defence s exponenciálním vážením času
  lineup delta: Σ (player value) chybějících vs náhradníků  (fotbal: xG/xA + minuty; hokej: GSAx gólman, TOI; e-sport: HLTV rating stand-in)
  bench depth: rozptyl kvality 12.–18. hráče
  wave: forma (vážená), rest days, tlak (tabulková situace), drama (late goals), coach change
  market: implikovaná p z Pinnacle/Betfair (bez marže) jako prior
   ↓
[core] bivariate Poisson / Dixon-Coles → matice P(h,a) (fotbal); per-period Poisson + OT/SO (hokej); Bradley-Terry per map + Bo3/Bo5 kombinace (e-sport)
   ↓
[calibration] isotonic / Platt na out-of-sample; report Brier, log-loss, reliability diagram
   ↓
[markets] odvození všech trhů z distribuce + „vysvětlení" (SHAP-like příspěvky vlny)
   ↓
[eval] CLV vs Pinnacle closing; flat-stake ROI; Kelly simulace
```

### 3.2 Zásady
- **Prior = trh.** Model začíná z implikované pravděpodobnosti Pinnacle a *upravuje* ji o sestavu a vlnu. Cíl není „porazit trh od nuly", ale být rychlejší/lepší v momentech, kdy se mění informace (potvrzený gólman, zveřejněná sestava T-60 min, stand-in).
- **Vlna má regularizaci.** Pokud se koeficient momenta nefituje signifikantně, UI ho ukazuje jako „náladu", ale do pravděpodobnosti jde ~0.
- **Každé číslo má datum a zdroj.** Uživatel vidí, kdy byla sestava potvrzena.

### 3.3 Backtest protokol (M1)
- Data: 3+ sezony Premier League/Bundesliga (Sportmonks), 2 sezony NHL, 1 sezona CS2 (HLTV/odds historie).
- Walk-forward po kolech, bez leaku (lineup data pouze z času T-1h).
- Metriky: Brier, log-loss, CLV (medián i průměr), ROI při flat stake nad hranou ≥ 3 %, počet sázek.
- Baseline: (a) jen trh, (b) Dixon-Coles bez sestavy, (c) plný model. Reportovat delta každé vrstvy.

---

## 4. Produkt (Flutter)

### 4.1 Obrazovky
1. **Dnes** — karty zápasů, u každé mini „dvě vlny" a A/B kolo.
2. **Zápas** — velká animace vln, přepínač trhů (1X2 / přesný výsledek / hendikep / třetina…), každý trh = pravděpodobnost + férový kurz + nejlepší kurz na trhu (z feedu) + jedna věta proč.
3. **Soupiska** — kdo chybí, náhradník, dopad v % (barevná lišta).
4. **Můj tip** — tipy za body, série, liga přátel (žádné peníze).
5. **(A-režim) „Vsadit u…"** — tlačítko s affiliate odkazem jen na whitelistované CZ sázkovky, s povinným 18+ a varováním o závislosti (§5j z. 40/1995), geo-gate CZ/SK.

### 4.2 Monetizace
- Ol1n Pass / premium (bez reklam, bez trackingu — konzistentní s brandem): plná historie, notifikace „sestava potvrzena", e-sport modul.
- Affiliate revshare (režim A) — **pozor na konflikt s brandem „no tracking"**: affiliate link = tracking cookie třetí strany. Řešení: transparentní opt-in, žádný SDK, jen odkaz.
- B2B API (režim B): per-request / měsíční licence na signály (lineup delta, goalie confirm, stand-in alert).

---

## 5. Compliance checklist (CZ + stores)
- [ ] Žádná sázka s majetkovou hodnotou (body nejdou směnit, prodat, vyhrát za ně cenu) → mimo z. 186/2016.
- [ ] Reklama/affiliate **jen** na provozovatele ze seznamu MF (whitelist); každá zmínka = 18+ + viditelné varování o riziku závislosti; žádné „zaručená výhra", žádný dojem zdroje příjmu (z. 40/1995 §5j; dozor Celní správa).
- [ ] Žádná propagace Polymarket/Kalshi/Stake/Cloudbet uživatelům v ČR.
- [ ] Apple 5.3: appka není real-money gaming (nic nepřijímá) → běžná kategorie; věkový rating 17+ (gambling-related content); nepoužívat IAP na nic, co souvisí s reálnou sázkou.
- [ ] Google Play: hazard je v CZ „allowed with limitations" — i pro affiliate/odds appky vyplnit gambling formulář, IARC rating, geo-omezení na CZ/SK; ověřit aktuální znění policy před releasem.
- [ ] GDPR: žádné 3rd-party trackery (brand), minimální data.
- [ ] Odds data: respektovat ToS feedů; scraping Tipsportu/Flashscore jen pro interní backtest, ne pro produkci.

---

## 6. Milníky

| M | Cíl | Exit kritérium | Odhad |
|---|---|---|---|
| **M0 Validace dat (hokej)** | API-HOCKEY 3 sezony (Allsvenskan, Maxa/Chance liga, Tipos extraliga, Extraliga, NHL) + OddsPapi historie kurzů + gólmani ze swehockey/ceskyhokej; NHL jako kontrola | tabulky `games/goalies/odds_snapshots` naplněné, join klíče ≥ 98 %, `DATA_M0.md` s pokrytím, marží a výběrem 2 lig do M1 | 1 týden |
| **M1 Model + backtest** | Kalman rating + dlouhá/krátká vlna (+ lineup delta kde jsou data); walk-forward per liga; FDR; simulace žebříku 1/5/10 | **Flat ROI > 0 na ≥ 500 singlech AND CLV > 0 vs medián knih na ≥ 1 lize přes 2 sezony**; žebřík reportován v log-růstu | 2–3 týdny |
| **M2 Režim C test** | 4 týdny papírově, pak mikro vklady na vlastním účtu u CZ sázkovky (legální); denní ledger od prvního dne | CLV > 0 na ≥ 200 sázkách; ROI reportováno s intervalem | 1 měsíc paralelně |
| **M3 Flutter MVP** | Dnes/Zápas/Soupiska/Můj tip, bez affiliate; fotbal top 5 lig + NHL | TestFlight + interní test, 20 uživatelů, retence D7 ≥ 25 % | 3 týdny |
| **M4 Režim A/B** | Tipsport Partners žádost, compliance review, B2B pilot s Betano/menší sázkovkou | podepsaný 1 affiliate + 1 B2B rozhovor s daty | průběžně |

Go/no-go po M1 — Director nemá pokračovat na M3, pokud M1 nedopadne (ušetří se nejvíc času).

---

## 7. Oponentura — proč to nemůže fungovat

| # | Argument proti | Síla | Co by ho vyvrátilo |
|---|---|---|---|
| 1 | **Trh je efektivní.** Closing line sharp knih (Pinnacle) obsahuje sestavy, zranění i gólmany. Indie model s veřejnými daty ji dlouhodobě neporazí. | vysoká | CLV > 0 v M1/M2 na stovkách sázek; realisticky jen v oknech „nová informace" |
| 2 | **Momentum neexistuje tak, jak ho fanoušek cítí.** Ötting et al.: sázkaři na momentum sázejí víc, ale kurzy ani výsledky to nemění; sázet na něj prodělává. „Vlna" tedy prodává iluzi. | vysoká | Fitovaný koeficient vlny signifikantně ≠ 0 mimo sestavu; jinak vlnu degradovat na UI narativ |
| 3 | **Data o absencích jsou pozdě nebo drahá.** Oficiální sestavy ~60 min před výkopem — trh je zapracuje za sekundy. Dřívější „expected lineups" jsou add-on a nejsou spolehlivé. Hokejová Extraliga = Sportradar enterprise. | vysoká | Rychlost ingest → notifikace < 30 s; e-sport rostery jsou veřejné dny dopředu |
| 4 | **Legálně to v ČR nemůže být sázecí appka.** Bez licence MF nic nepřijmeš; propagovat smíš jen whitelist; krypto a predikční trhy nesmíš uživatelům ani ukázat. Z „sázecí appky" zbyde tipovačka s affiliate odkazy — to už mají Tipsport (Tip Liga), Flashscore, Forebet. | vysoká | Diferenciace = kvalita vysvětlení + e-sport + soupiska; ne „sázení" |
| 5 | **Affiliate model je v rozporu s brandem.** „Bez reklam, bez trackingu" vs. revshare z prohraných peněz uživatelů (NGR = jejich ztráta). Reputační riziko + etika. | střední | Premium místo affiliate; nebo B2B |
| 6 | **Store riziko.** Apple i Google hází gambling-related appky do nejpřísnější kategorie; jedna chyba v geo-gatingu = removal, který může ohrozit celý org účet (Ol1n org s dalšími appkami). | střední | Publikovat pod odděleným účtem? Ověřit policy před M4; bez affiliate nejde o gambling |
| 7 | **Náklady na data rostou s ambicí.** Fotbal top ligy levně (€29–179/měs.), ale Extraliga, e-sport soupisky, live odds streaming = stovky až tisíce USD/měs. | střední | Začít NHL + PL + CS2 (vše dostupné levně) |
| 8 | **Scraping Tipsportu/Flashscore** je ToS-breaking a Livesport (Praha) si to hlídá. | střední | Jen feedy s licencí |
| 9 | **Retence tipovaček je mizerná** bez peněz v sázce; Forebet má miliony instalací, ale na iOS stovky MAU. | střední | Sociální liga přátel + push „sestava potvrzena" = důvod otevřít |
| 10 | **Jeden člověk, 80 projektů.** Model vyžaduje průběžnou údržbu (změny API, nové sezony, drift). | vysoká | Striktní go/no-go po M1; Director automatizuje ingest + týdenní report |
| 11 | **Okrajová neefektivita je často šum.** Winkelmann et al. 2023: edge v jedné sezoně/lize se objeví i při plně efektivním trhu; nepřežívá replikaci. | vysoká | FDR korekce, ≥ 3 sezony, replikace v živé sezoně před nasazením |
| 12 | **Limity a omezení účtu stropují profit.** V nižších ligách CZ knihy přijmou tisíce Kč na tiket; vítězné účty dostanou limit. Realistický strop = desítky tisíc Kč/rok, ne byznys. | vysoká | Více účtů není cesta (porušení podmínek). Jediná škála je B2B prodej signálů nebo burza (Betfair) mimo CZ |
| 13 | **V nižších ligách nejsou data o sestavách** → hlavní USP („vliv absentujícího hráče") tam nefunguje; zbývá dlouhá vlna, kterou výzkum momenta zpochybňuje. | vysoká | Kalman „rychlost" musí mít signifikantní koeficient; jinak zpět k top ligám/e-sportu |
| 14 | **Kombinace 10 zápasů = produkt s nejvyšší marží sázkovek.** „Maximální profit" přes dlouhé akumulátory je přesně chování, na kterém knihy vydělávají; bez edge na každé noze je to −40 až −50 % EV. | vysoká | Jen Kelly žebřík s vkladem ≤ 1/100 na 10-kombi; smysl pouze s kombi bonusem > kumulovaná marže |

**Nejpravděpodobnější scénář selhání:** M1 ukáže Brier ≈ trh, CLV ≈ 0 → produkt je hezká vizualizace bez informační výhody; affiliate příjem malý; appka zůstane v portfoliu jako „zajímavý experiment".
**Nejpravděpodobnější scénář úspěchu:** e-sport (stand-in) + NHL (gólman) signály jako **B2B feed / premium notifikace**, fotbal jen jako UX vitrína.

---

## 8. SWOT konkurenčních platforem

### Tipsport (+Chance)
- **S:** dominance, komunita (Tip Liga, fórum), TV Tipsport streamy, nejširší nabídka, e-sport.
- **W:** nižší kurzy (marže), UX zastaralé, appka mimo Play Store (sideload), negativní reputace u části hráčů.
- **O:** B2B nákup externích signálů/UX vrstev; affiliate partner s nejvyšší konverzí.
- **T:** Betano a zahraniční brandy po uvolnění registrace; černý trh; návrhy na omezení reklamy.

### Fortuna (FEG)
- **S:** druhá značka, multi-trh CZ/SK/PL, naming ligy, widoké affiliate rozpětí (až 70 %).
- **W:** zaostává za Tipsportem v šíři, UX; stížnosti na live sázení.
- **O:** affiliate partner č. 2; sponzoring sportu.
- **T:** stejné jako Tipsport + ztráta podílu vůči Betano.

### Betano
- **S:** nejlepší UX/aplikace, globální tech (Kaizen), agresivní marketing (Sparta), růst.
- **W:** nová značka, užší nabídka, affiliate s negative carryover.
- **O:** **nejotevřenější B2B kandidát** — hledá diferenciaci proti dvěma zavedeným.
- **T:** regulace, cena akvizice zákazníka v CZ.

### bet365 / Pinnacle / Betfair Exchange
- **S:** globální likvidita; Pinnacle = sharp benchmark; Betfair = burza bez marže.
- **W:** v CZ nelicencované → nepropagovatelné; API jen pro zákazníky/agregátory.
- **O:** zdroj „pravdy" (closing line, implied p) pro kalibraci a CLV.
- **T:** žádná pro nás — nejsou konkurent, jsou měřítko.

### Polymarket / Kalshi (predikční trhy)
- **S:** binární trhy = přirozené „A/B"; sport dominuje objemu 2026; otevřená API, hloubka orderbooku.
- **W:** v EU/CZ bez licence; KYC/geo; regulace v pohybu (NFL varování, Nevada soudy, návrh zákazu sportu).
- **O:** režim C — vlastní test edge; aggregátorová data zdarma.
- **T:** právní riziko při jakékoli propagaci uživatelům v ČR.

### Stake / Cloudbet / Thunderpick (krypto)
- **S:** hloubka trhů, vysoké limity, instant výběry, Thunderpick = nejhlubší e-sport.
- **W:** Curaçao, bez CZ licence, spory o výběry.
- **O:** jen datový zdroj kurzů (přes OddsPapi/odds-api.io).
- **T:** reklama na ně v ČR = správní delikt.

### Livesport / Flashscore / Betexplorer / Oddsportal (Praha)
- **S:** obří traffic v CZ, vlastní kurzová srovnání, historie kurzů, lokální firma s právníky.
- **W:** generický produkt, bez modelu „proč", bez emocionální vrstvy.
- **O:** potenciální partner/kupující pro B2B signály (jsou v Praze).
- **T:** mohou kdykoli přidat „predikce" jako feature; ToS vůči scrapingu.

### Forebet / Betegy / GoalGPT / Sofascore (predikční appky)
- **S:** zavedené, Forebet 3,4 M instalací Android, matematické modely od 2009; Sofascore má xG + „who will win".
- **W:** slabá retence (Forebet iOS ~stovky MAU), reklamní model, bez sestavových vysvětlení, bez hokeje/e-sportu v CZ kontextu.
- **O:** mezera: vysvětlitelnost + soupiska + e-sport + bez reklam.
- **T:** AI predikční appky se množí (GoalGPT, betsAI…) — cesta „AI predikce" je přeplněná, cesta „sestava + vlna + vysvětlení" méně.

---

## 9. Otevřené otázky pro Ol1na
1. ~~Která vlna první?~~ **Rozhodnuto: hokej** (§1.4.5). Druhý sport později — CS2 (stand-in) nebo fotbal (FNL/League Two)?
2. Snese brand „bez trackingu" affiliate odkazy? Nebo rovnou premium + B2B?
3. Publikovat pod org účtem Ol1n (riziko pro ostatní appky), nebo zvlášť?
4. Chceš režim C (vlastní mikro-trading) vůbec zkoušet? Je to nejrychlejší pravdivý test, ale daňově/právně šedé.
5. Název: „Vlna" / „Tipánek" (ladí s Běhánek/Hajánek/Papánek) / jiný?

## 10. Co má Director udělat jako první
1. Založit repo `lioilsources/Vlna` (Go + Flutter monorepo), tento soubor do rootu.
2. M0-hokej podle §1.4.5: `cmd/ingest-hockey` (API-HOCKEY free key, OddsPapi free key, scraper swehockey/ceskyhokej s cache) → Postgres na JODA; cron snapshoty kurzů T-24h / T-1h / closing pro Allsvenskan + Maxa ligu + NHL; `DATA_M0.md`.
3. M1: Go job `cmd/model-hockey` — per-třetina Poisson + gólman delta + Kalman vlna; walk-forward per liga; FDR; žebřík 1/5/10 v log-růstu; `REPORT_M1.md` s ROI/CLV/Brier per liga a per vrstva (bez gólmana / s gólmanem / s vlnou).
4. Od prvního dne M1 zapisovat denní „Dnes" žebříček do ledgeru (papírově) — i když appka neexistuje.
5. Teprve po go/no-go: Flutter MVP (Dnes, Brankáři, Ledger).

---

## 11. Zdroje (ověřeno 11/2026)
- CZ trh: arecenze.cz srovnání 2026; lupa.cz (Betano/Tipsport/Sazka/Synot); e15.cz (online podíl Tipsportu 94 %); seznamzpravy.cz (zisk Tipsport 3,5 mld.); forbes.cz (Betano vstup po pádu registrace na pobočce).
- Affiliate: Tipsport Partners (whoaff/statsdrone), Fortuna Affiliates CZ (statsdrone), Betano Affiliates (statsdrone).
- Odds API: oddspapi.io (370 knih, free tier, historie), odds-api.io (250+, Polymarket/Kalshi depth, £99/měs.), the-odds-api.com.
- Data: docs.sportmonks.com (lineups, sidelined, expected lineups), api-sports.io, developer.sportradar.com (NHL, Global Ice Hockey vč. Extraligy), sportly (NHL API).
- E-sport: odds-api.io esports, oddspapi esports, sportsapis.dev (PandaScore/GRID ceny).
- Predikční trhy: sacra.com (Polymarket US 05/2026, sport), foxsports (CFTC swaps 2026), predictionhunt.com (API přehled), frontofficesports.com (regulační tlak).
- Krypto: cryptsy.com, cryptobriefing.com (Cloudbet 2026), reviews Thunderpick.
- Legislativa: z. 186/2016 Sb. §3 (epravo), MF stanovisko 2017, arws.cz (kvízové appky, affiliate = šiřitel reklamy), mf.gov.cz varování influencerům 2026, z. 40/1995 §2(1)(f), §5j; celnisprava.gov.cz metodika dozoru.
- Stores: Apple App Store Review Guidelines 5.3.4; support.google.com/googleplay (gambling apps, CZ allowed with limitations).
- Výzkum: Ötting, Deutscher, Singleton, De Angelis — Betting on Momentum in Contests, Economic Inquiry 2025 (arXiv 2211.06052); Winkelmann, Ötting, Deutscher, Makarewicz — Are Betting Markets Inefficient? JSE 2023 (neefektivity nejsou perzistentní); Lezana — Fear of the Draw, JSE 2026 (remízy/hosté podceňovány v EPL); Byrnes & Farinella (NBA momentum, 56,5 %); Scholtes & Karakuş — Bayes-xG; ESPN (gólman a linie NHL); pillarlab (starter vs backup GSAx).
- Kombi produkty CZ: Tipsport Maxikombi/Extratip/UNI sázka, Fortuna Multikombi/Favorit Plus (arecenze.cz, přehledy 2026) — kombi bonusy ověřit přímo u providerů před M2.
