# Plán pro Opus — „Vlna“ (pracovní název)

Pracuje se po fázích, v pořadí. Po každé fázi: testy zelené, lint čistý,
commit. Další fáze nezačíná, dokud nejsou splněná akceptační kritéria té
předchozí. **Fáze 1–3 nemají žádné UI.** Appka se staví až po bráně G1, a jen
pokud model prošel. Když neprošel, projekt končí a tenhle plán se archivuje
s výsledky backtestu — i to je výstup.

Před čtením plánu si přečti `03_OPONENTURA.md`. Plán je odpovědí na ni;
každá brána níže odkazuje na námitku, kterou má vyvrátit.

## 0. Rozhodnutí (zodpoví zakladatel, Opus se bez nich nehne)

| Otázka | Výchozí návrh | Proč |
|---|---|---|
| Co appka **je** | Analytický nástroj. Žádné přijímání sázek, žádné peníze uživatelů, žádný účet u sázkovky přes nás. | Přijímání sázek = licence MF, kapitál, kauce. Mimo rozsah navždy. |
| Sporty v MVP | Fotbal (Chance Liga, Premier League, Bundesliga, Serie A, La Liga) a hokej (Extraliga, NHL) | Fotbal má nejlepší data (sestavy, xG). Hokej je český trh a třetinové trhy. Nic dalšího, dokud model nefunguje. |
| Trhy v MVP | 1X2, dvojitá šance, remíza bez sázky, hendikep (evropský i asijský), over/under, přesný výsledek, poločas/třetina (vítěz, góly), poločas–konec, oba dají gól | Všechny plynou z jedné věci: z rozdělení skóre. Jedno jádro, mnoho trhů. |
| Sázkovky v MVP | Tipsport, Chance, Fortuna, Betano, Sazkabet (jen licencované v ČR) | Právně bezpečná množina. Viz `02_PROVIDERI_A_FIT.md`. |
| Referenční „ostrý“ kurz | Pinnacle přes agregátor (OddsPapi / Odds-API.io), fallback Betfair Exchange přes agregátor | Pinnacle API je od 23. 7. 2025 zavřené pro veřejnost. Betfair není v ČR licencovaný, ale jeho **cena** je veřejná informace a smíme ji použít jako referenci. |
| Monetizace | Předplatné (Pro). Affiliate **pouze** k licencovaným CZ operátorům, a až ve fázi 7. | Affiliate k nelicencovaným = reklama na nelegální hazard (metodika CS MI‑2025‑023). |
| Backend | Ano. Python (model, ingest, scheduler), Postgres + Timescale, hostováno v EU. | Model potřebuje historii, backtest a cron. On‑device to nejde. |
| Mobil | Flutter, Riverpod, drift (stejný stack jako Been Here). iOS i Android ze stejného kódu. | Autor ho zná. |
| Repo | Nové repo `vlna` (monorepo: `model/`, `api/`, `app/`). | Been Here je privacy‑first on‑device appka; mixovat ji s betting backendem je špatně po všech stránkách. |
| Název | TBD. Ne „sázka“, „bet“, „tip“ v názvu (store review, reklama). | |

## 1. Produkt

### Co uživatel vidí

1. **Zápas** — dvě vlny vedle sebe. Každá je jeden obrázek: křivka
   posledních 10 zápasů (výkon, ne výsledek), pod ní tři ikony: kdo chybí a
   kolik procent „síly“ to je, kdo ho nahradí a kolik jí vrátí, hloubka
   lavičky. Vpravo náš odhad: 1X2, očekávané góly, rozdělení skóre.
   Druhá záložka **Srovnání**: oba týmy vedle sebe, bloky metrik a karty
   „Co rozhoduje“ z dekompozice odhadu; pravidla v `07_SROVNANI_TYMU.md`.
2. **Trhy** — tabulka: trh, náš odhad pravděpodobnosti, kurz u každé
   uživatelovy sázkovky, implikovaná pravděpodobnost po odečtení marže,
   hrana (edge) v procentních bodech, doporučená velikost (frakční Kelly,
   stropovaná). Zelené jen to, co má hranu nad prahem **a** zároveň projde
   CLV filtrem (viz 4.3).
3. **„Chci sázet X“** — uživatel řekne, co chce (např. 0:1, Sparta vyhraje
   druhou třetinu). Appka nehodnotí, jestli je to rozumné. Řekne: naše
   pravděpodobnost, férový kurz, kurz u tvých sázkovek, kolik ti to stojí
   ročně, kdybys to dělal pořád. **Toto je ta „emocionální“ část produktu:
   sázkař se rozhodl, my mu dáme číslo, ne kázání.**
4. **Můj deník** — každá sázka, kterou uživatel zaznamená (ručně nebo
   screenshotem tiketu; OCR až později), náš odhad v čase sázky, uzavírací
   kurz, CLV. Po 50 sázkách první zpětná vazba: „Tvoje sázky na přesný
   výsledek mají CLV −6 %. Tvoje hendikepy +1,5 %.“ Toto je jediná poctivá
   metrika, kterou můžeme uživateli dát, a je to jádro retence.
5. **Nastavení** — moje sázkovky, moje ligy, práh hrany, Kelly frakce,
   limity (denní strop na počet doporučení; vyplý „night mode“ 00–06).
6. **Oblíbený tým** — kádr, lavička, pětky a dvojice, trenéři, historie,
   stadiony a doma/venku. Fanouškovská vrstva; co z ní smí do modelu, je
   v `06_DETAIL_TYMU.md`.

### Co appka nikdy nedělá

Nepřijímá sázky. Nedrží peníze. Neposílá push „teď si vsaď“. Nenabízí
bonusy. Neukazuje kurzy operátorů bez české licence. Nemá leaderboardy
výher. Nemá „jistoty“, „tutovky“, „free tipy“.

### Mimo MVP (nerozbít architekturou)

Live (in‑play) model, e‑sports, predikční trhy, OCR tiketů, sdílení,
komunita, automatické sázení (nikdy), web verze.

## 2. Model

### 2.1 Vlna (team state)

Pro tým `T` před zápasem `m`:

- **Výkonová řada** `perf[t]` za posledních N=10 zápasů: ne výsledek, ale
  rozdíl očekávaných gólů (xG − xGA) upravený o sílu soupeře a o
  domácí/venkovní prostředí. Hokej: rozdíl očekávaných gólů z pokusů (Corsi/
  Fenwick vážené polohou, kde to data dovolí; jinak střely na branku vážené
  nebezpečností).
- **Vlna** = exponenciálně vážený průměr `perf` s poločasem ~4 zápasy, plus
  *trend* (sklon regresní přímky přes posledních 5) a *rozptyl*. Tři čísla:
  úroveň, směr, stabilita. „Vlna“ v UI je obrázek těchto tří čísel.
- **Absence**: pro každého hráče `p` jeho *podíl síly* `w[p]` = kombinace
  (a) podílu odehraných minut v posledních 20 zápasech, (b) on/off rozdílu
  xG týmu s ním a bez něj (regularizovaný; málo dat → smrsknout k nule),
  (c) externí hodnocení (rating z poskytovatele dat, pokud ho máme).
  Dopad absence = `w[p]` − `w[náhradník]`.
- **Lavička a střídání**: hloubka = součet `w` pro hráče, kteří obvykle
  střídají; rotace = počet změn v základní sestavě vůči minulému zápasu;
  únava = dny od posledního zápasu a počet zápasů za 14 dní. Hokej: 4 lajny,
  čas na ledě po lajnách, brankář (startující brankář je největší jeden
  faktor v hokeji; model ho má jako samostatný vstup s vlastním `w`).
- **Režimové změny**: nový trenér (reset vlny na průměr ligy, zvýšený
  rozptyl), přestupové okno.

### 2.2 Výsledkový model

- Fotbal: **Dixon–Coles** (bivariátní Poisson s korekcí pro 0:0, 1:0, 0:1,
  1:1) jako základ. Útok/obrana týmu = dlouhodobá síla (odhad přes celou
  sezonu s časovým vážením) **+ příspěvek vlny + příspěvek absencí**.
  Výstup: matice pravděpodobností skóre 0–7 × 0–7. Z ní vše ostatní.
- Poločasy: skórovací intenzita rozdělená na poločasy podle empirického
  poměru ligy (zhruba 45/55) a upravená o tempo týmu; poločas–konec z
  podmíněných pravděpodobností.
- Hokej: Poisson po třetinách s ligovým profilem intenzity (třetí třetina
  má víc gólů; prázdná branka modelovaná zvlášť), prodloužení a nájezdy jako
  samostatný krok (3‑bodový systém Extraligy vs. NHL). Výstup: rozdělení
  skóre po třetinách → vítěz třetiny, góly ve třetině, celkový vítěz
  v základní hrací době vs. včetně prodloužení, hendikep −1,5/+1,5.
- Kalibrace: Platt/izotonická regrese na odložené sezoně. Reportovat Brier,
  log‑loss, kalibrační křivku, **vždy vedle stejné metriky pro odmaržovaný
  kurz Pinnacle** (benchmark). Model, který nemá menší log‑loss než
  Pinnacle, nemá hranu, ať vypadá jak chce.

### 2.3 Hrana a velikost sázky

- Odmaržování kurzů: Shinova metoda (ne prosté proporcionální), protože
  marže sedí nerovnoměrně na outsiderech (favourite‑longshot bias).
- `edge = p_model − p_implied`. Doporučení jen při `edge > práh` (výchozí
  3 p. b. pro 1X2, 5 p. b. pro přesný výsledek a třetiny, kde je marže
  vyšší a náš model slabší).
- Velikost: frakční Kelly (výchozí 1/4), strop 2 % bankrollu, strop 5
  doporučení na den na uživatele.
- **CLV filtr**: trh, na kterém náš model historicky nedosahuje kladného
  CLV, se nezelená, i když ukazuje hranu. Toto je pojistka proti
  přefitování.

## 3. Architektura

```
vlna/
  model/        Python 3.12, polars, numpy, scipy, statsmodels, pymc (jen pokud bude třeba)
    ingest/     adaptéry: sportmonks, apisports, enetpulse|sportradar, oddsapi_io, oddspapi
    features/   vlna, absence, lavička, únava
    models/     dixon_coles, hockey_periods, calibration
    backtest/   walk-forward harness, CLV, Brier, log-loss, PnL simulace
    cli/        `vlna backtest --sport football --from 2021 --to 2025`
  api/          FastAPI, Postgres+Timescale, APScheduler; read-only pro appku
  app/          Flutter, Riverpod, drift (cache + deník offline)
  docs/
```

Pravidla:

- `model/` nezná FastAPI ani Flutter. Vstupy a výstupy jsou parquet/
  dataclassy. Testovatelné bez sítě (fixtures = zmražené odpovědi API).
- Každý adaptér v `ingest/` má **jeden** úkol: převést cizí formát na náš
  kanonický (`Match`, `Lineup`, `PlayerMinutes`, `OddsSnapshot`). Výměna
  poskytovatele = nový adaptér, nic víc.
- Kurzy se **snapshotují** (každých 15 min předzápas, 1 min poslední hodinu)
  a nikdy nepřepisují. Bez historie kurzů není CLV a bez CLV není nic.
- Appka je tenký klient. Žádný model na telefonu. Deník sázek je lokální
  (drift) s volitelnou synchronizací; uživatel může appku používat bez
  účtu.
- Žádná třetí‑stranová analytika v appce. Vlastní události do našeho API,
  anonymní ID.

## 4. Data a poskytovatelé (konkrétně)

| Potřeba | Fotbal | Hokej | Poznámka |
|---|---|---|---|
| Zápasy, výsledky, xG, střely | Sportmonks (Growth, od ~€79/měs ročně) nebo API‑Football (Ultra $29/měs, bez xG na všech ligách) | Enetpulse nebo Goalserve (Extraliga + NHL); Sportradar Global Ice Hockey jen pokud rozpočet dovolí | xG pro Chance Ligu je slabé místo; zvážit vlastní xG z polohy střel, pokud poskytovatel dává souřadnice |
| Oficiální sestavy | Sportmonks (≈60 min před výkopem, ve všech plánech) | Enetpulse/Goalserve lineups tier | |
| Očekávané sestavy | Sportmonks Premium Expected Lineups, €199/měs add‑on | Nic rozumného; odvodit z posledních sestav + zprávy o zraněních | Pozdní změny add‑on nechytá — model s tím musí počítat (šířka intervalu) |
| Zranění/tresty | Sportmonks injuries (Growth+) | Ruční kurace pro Extraligu v MVP, NHL z veřejných zdrojů | |
| Kurzy CZ sázkovek | Odds‑API.io (Tipsport od 12/2025; Fortuna ověřit), jinak **nic** — scraping Tipsportu/Fortuny je porušení podmínek a křehké | stejné | Nejdřív ověřit pokrytí přímo dotazem na `/bookmakers`. Toto je kritická závislost; bez ní appka neumí „kurz u tvé sázkovky“ |
| Referenční kurzy | OddsPapi nebo Odds‑API.io: Pinnacle, Betfair, bet365 | stejné | |
| Historie kurzů pro backtest | Odds‑API.io historical, football‑data.co.uk (zdarma, uzavírací kurzy top lig od 2000), Apify: OddsPortal (otevírací i uzavírací kurzy, 20 sezon, všechny trhy) a Flashscore (xG od 2023/24) scrapery, viz `05_APIFY_SCRAPERY.md` | OddsPortal přes Apify má i hokej | Apify jen pro jednorázový výzkumný sběr, ne pro produkci |

Rozpočet na data v MVP: 300–500 €/měs. Před prvním předplatitelem.

## 5. Fáze

### Fáze 0 — Rozhodnutí a kostra (1 týden)

Zakladatel zodpoví sekci 0. Opus: nové repo, `model/` s prázdnými balíčky,
CI (ruff, mypy, pytest), `.env.example`, kanonický datový model, dokument
`docs/DATA_CONTRACT.md`. Žádný kód modelu.

Akceptace: `make check` zelené; kanonický model má testy serializace.

### Fáze 1 — Datová páteř a backtest harness (3 týdny)

- Adaptéry pro jeden fotbalový zdroj a jeden kurzový zdroj. Stáhnout 4
  historické sezony 5 lig (≈ 7 000 zápasů) + uzavírací kurzy
  (football‑data.co.uk stačí na start).
- Walk‑forward harness: trénink do data `d`, predikce zápasů v `(d, d+7]`,
  posun. Výstup: tabulka predikcí s `p_model`, `p_pinnacle_devig`,
  výsledek, kurz.
- Metriky: Brier, log‑loss, kalibrace, CLV, simulovaný PnL při frakčním
  Kelly s praktickými limity (max sázka, marže).
- **Benchmark baseline**: holý Dixon–Coles bez vlny a bez absencí. Vše
  další se měří proti němu a proti Pinnacle.

Akceptace: harness běží end‑to‑end nad fixtures bez sítě; baseline má
log‑loss do 2 % od odmaržovaného Pinnacle (to je známý výsledek, slouží jako
kontrola, že harness nemá chybu).

### Fáze 2 — Vlna a absence (3 týdny)

Features podle 2.1. Ablace: baseline; + vlna; + absence; + lavička/únava;
+ vše. Každá přidaná složka musí **zlepšit log‑loss na odložených sezonách**,
jinak jde ven. Zvlášť reportovat, zda „vlna“ zlepšuje predikci, nebo jen
kopíruje trh (korelace rezidua modelu s pohybem kurzu od otevření
k uzavření — pokud vysoká, vlna jen opakuje to, co trh už ví).

Akceptace: ablační tabulka v `docs/RESULTS_PHASE2.md`. Rozhodnutí, které
složky zůstávají.

### Fáze 3 — Trhy, kalibrace, hokej (3 týdny)

Rozdělení skóre → všechny trhy z 0. Kalibrace per trh. Dekompozice
příspěvků vstupů k odhadu (`contributions`, `07_SROVNANI_TYMU.md` §4) jako
součást výstupu modelu; slouží i ablacím. Hokejový model na
NHL (data jsou) s ověřením na Extralize (data jsou horší; pokud nejsou,
Extraliga odchází z MVP, ne naopak).

**Brána G1 (go/no‑go, odpovídá námitkám 1, 2, 3, 5 oponentury):**

| Kritérium | Práh |
|---|---|
| Log‑loss modelu na 1X2 vs. odmaržovaný Pinnacle, 2 odložené sezony, ≥ 3 000 zápasů | model ≤ Pinnacle − 0,5 % **nebo** model s Pinnaclem jako featurem ≤ Pinnacle − 0,3 % |
| CLV sázek s `edge > 3 p. b.` proti uzavíracímu Pinnacle | > 0 s 95 % intervalem spolehlivosti nad nulou; ≥ 1 500 sázek |
| Simulovaný yield při Kelly 1/4 u **CZ kurzů** (ne Pinnacle), včetně 5 % marže | > 0 na obou odložených sezonách |
| Ablace | vlna + absence zlepšují log‑loss proti holému DC ve všech 5 ligách |

Jestli projde jen první řádek, ale ne CLV: model „ví“, ale nedá se to vsadit.
Produkt se mění na čistě informační (bez „zelených“ hran), nebo končí.
Jestli neprojde nic: konec, archivace, výsledky do `docs/POSTMORTEM.md`.

### Fáze 4 — Živé kurzy a API (2 týdny)

Snapshoty kurzů CZ sázkovek + reference. FastAPI read‑only: `/matches`,
`/matches/{id}/waves`, `/matches/{id}/markets`, `/journal` (auth). Denní
cron: stáhnout zápasy, sestavy, přepočítat. Monitoring zpoždění dat
(sestava přišla kdy vs. výkop).

Akceptace: pro dnešní zápasy 5 lig jsou do 30 s od přijetí sestavy
spočítané všechny trhy; snapshoty kurzů bez děr přes 24 h.

### Fáze 5 — Appka MVP (4 týdny)

Obrazovky 1 (včetně záložky Srovnání v rozsahu `07_SROVNANI_TYMU.md` §7),
2, 3, 5 ze sekce 1 a zjednodušený Oblíbený tým (Přehled, Kádr, Trenér;
`06_DETAIL_TYMU.md` §7). Deník ručně (obrazovka 4) bez OCR.
Lokalizace cs/en. Věková brána 18+, povinné upozornění podle § 5j zákona
o regulaci reklamy (text „Ministerstvo financí varuje: Účastí na hazardní
hře může vzniknout závislost“ + kontakt na pomoc) na každé obrazovce
s kurzy — i když nejsme provozovatel, chceme být nad čarou, ne na ní.

Akceptace: TestFlight; 10 uživatelů‑přátel používá 4 týdny; zaznamenají
≥ 200 sázek do deníku; appka ani jednou neukáže kurz operátora bez CZ
licence (test).

### Fáze 6 — Deník jako produkt (3 týdny)

CLV zpětná vazba uživateli, rozpad podle trhů, „kolik tě stojí emoce“
(rozdíl mezi tím, co vsadil, a tím, co by vsadil podle modelu). Limity a
„ticho“ (bez doporučení) jako funkce, ne jako omluva. Plný Oblíbený tým:
pětky a brankářské dvojice, historie a trofeje, stadiony a tour, grafy
(`06_DETAIL_TYMU.md` §2.3–2.6).

Akceptace: retence D30 u testovací skupiny > 40 %. Jinak se produkt
nemonetizuje, ale přepracuje.

### Fáze 7 — Monetizace (2 týdny)

Předplatné Pro (RevenueCat). Zdarma: vlny, odhad 1X2, deník. Pro: všechny
trhy, hrany, CLV. Affiliate k CZ licencovaným operátorům **pouze** jako
„otevřít u Tipsportu“ u konkrétního trhu, s označením reklamy a s upozorněním
podle zákona. Právní revize před spuštěním (hodina s právníkem na hazard).

### Fáze 8 — Rozšíření (až po 6 měsících provozu)

Podle `02_PROVIDERI_A_FIT.md`: nejdřív NHL do hloubky, pak e‑sports CS2
(data Oddin/Abios jsou drahá; ověřit), live model (vlna *během* zápasu po
střídání — to je technicky nejzajímavější, ale datově nejdražší). Predikční
trhy a krypto sázkovky **ne**, dokud se nezmění regulace (viz 02).

## 6. Pracovní pravidla pro Opus

- Nikdy neukazuj metriku modelu bez metriky trhu vedle. Osamocené „75 %
  úspěšnost“ je zakázaný výstup.
- Žádné odložené sezony se nedotýkej při ladění. Když je potřeba víc dat na
  ladění, přidej starší sezony, ne novější.
- Každá fáze končí dokumentem `docs/RESULTS_PHASE{n}.md` s tabulkou a jednou
  větou závěru. Zakladatel čte jen tu větu a tabulku.
- Ptej se zakladatele jen na věci ze sekce 0 a na brány. Všechno ostatní
  rozhodni sám a zapiš do `docs/DECISIONS.md`.
- Když se poskytovatel dat ukáže jako nepoužitelný (pokrytí, cena, ToS),
  nepřecházej potichu na scraping. Zapiš, zastav fázi, navrhni alternativu.
- Kód modelu je v Pythonu; ne Dart. Appka model nikdy nepočítá.
- Scraping webů sázkových kanceláří je mimo. Tečka. Čtení agregátorů
  (OddsPortal, Flashscore, Sofascore) přes Apify je dovolené jen pro
  výzkumná data do backtestu ve fázích 1–3; pravidla v `05_APIFY_SCRAPERY.md`.

## 7. Rozpočet času a peněz

| Fáze | Týdny | Peníze/měs |
|---|---|---|
| 0–3 (model, bez UI) | 10 | 150–350 € data |
| G1 | — | — |
| 4–7 | 11 | 300–500 € data + hosting ~40 € |
| Celkem do prvního předplatitele | ~21 týdnů | ~3 000 € |

Break‑even při 9 €/měs Pro a 30 % konverzi z aktivních: cca 150
předplatitelů. Pro českou nišu „sázkař, který chce čísla a ne tipy“ je to
dosažitelné jen s organickým dosahem (fóra, YouTube rozbor zápasu), protože
placená reklama na hazardní téma je prakticky nemožná (viz 02 a 03).
