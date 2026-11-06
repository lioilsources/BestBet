# Detail oblíbeného týmu — specifikace

Zadání (2026-11-05): kádr a lavička, hráči, manažeři a trenéři s historií,
úspěchy a statistikami, grafy výkonu v průběhu let, góly, nahrávky, +/−,
hokejové pětky, obranné dvojice, brankářské dvojice, úspěšnost na
jednotlivých stadionech, domácí zápasy vs. „venkovní tour“.

Tohle je **fanouškovská vrstva** appky. Je to to, co uživatel otevře každý
den i bez zápasu, a je to místo, kde se vlna, absence a lavička z modelu
(`01_PLAN_PRO_OPUS.md`, §2.1) stávají viditelnými věcmi, ne čísly. Zároveň
je to past: většina z toho je pro predikci šum a nesmí to do modelu
prosakovat jinak, než je níž napsané.

## 1. Dvě role každé položky

Každá informace na obrazovce týmu má dvě role a ta druhá je důležitější
hlídat:

| Položka | Pro fanouška | Pro model |
|---|---|---|
| Kádr, lavička, zranění | kdo hraje, kdo chybí | vstup absencí (`w[p]`) a hloubky lavičky — **už je v modelu** |
| Pětky / lajny, obranné dvojice | kdo s kým hraje | společný čas na ledě a xGF% dvojic/trojic zpřesňuje `w[náhradník]` — **do modelu jen jako regularizovaný příspěvek** |
| Sehrané dvojice a WOWY (spolu / odděleně / partner chybí) | „bez Nečase je Pastrňák poloviční“ | **ano, jako člen `chem[A,B]`** v dopadu absence (§2.3); regularizovaný k nule při malém vzorku |
| Brankářská dvojice | kdo chytá dnes | startující brankář je největší jeden faktor v hokeji — **už je v modelu** |
| Trenér, historie, úspěchy | příběh | jen „změna trenéra“ jako režimový reset — **nic víc** |
| Grafy výkonu po sezonách | křivka klubu | **ne**; vlna bere posledních 10 zápasů, ne 10 let |
| Góly, nahrávky, +/− hráče | hvězdy | nepřímo přes `w[p]`; samotné +/− je slavně šumová metrika |
| Úspěšnost na stadionu X | „v Třinci nikdy nevyhrajeme“ | **ne**. Při 1–3 zápasech ročně na jednom stadionu je to čistý malý vzorek. Ukazujeme, ale nepouštíme do modelu. |
| Doma / venku, venkovní tour | bilance | domácí výhoda je parametr ligy (ne týmu) + únava z cestování (dny, počet venkovních zápasů v řadě) — **už v modelu** |

Pravidlo: obrazovka týmu **čte** z datového modelu, do modelu **nezapisuje**
nic nového kromě toho, co je v tabulce výše označené jako „do modelu“.

## 2. Obrazovky

Tým má šest záložek. Fotbal a hokej sdílí layout, liší se obsahem třetí.

### 2.1 Přehled

- Hlavička: logo, liga, pozice v tabulce, forma (posledních 5 jako
  barevné tečky), **vlna** (stejný obrázek jako na zápase: úroveň, směr,
  stabilita).
- Příští zápas s odhadem 1X2 a odkazem na detail zápasu.
- Tři karty: „Kdo chybí“ (hráč, důvod, odhad návratu, % síly týmu),
  „Dnešní brankář“ (hokej) / „Očekávaná sestava“ (fotbal), „Trenér“ (jméno,
  od kdy, bilance).

### 2.2 Kádr a lavička

- Seznam hráčů seskupený podle pozice (fotbal: G/O/Z/Ú; hokej: brankáři,
  obránci, útočníci), s číslem, věkem, národností, smlouvou do, tržní
  hodnotou (fotbal, Transfermarkt), statusem (fit / zraněn / trest /
  nejistý).
- U každého hráče sezonní řádek: zápasy, start/střídání, minuty (hokej:
  čas na ledě), góly, nahrávky, body, +/− (hokej), xG a xA (fotbal, kde
  jsou), rating poskytovatele.
- Přepínač „lavička“: hráči, kteří v posledních 10 zápasech nastoupili
  jako střídající nebo vůbec. U nich podíl minut a `w[p]`, tedy kolik síly
  tým ztratí, když musí nastoupit místo jedničky.
- Detail hráče: kariérní tabulka po sezonách a klubech, graf gólů +
  nahrávek na zápas po sezonách, graf minut po sezonách, historie zranění,
  úspěchy (tituly, individuální ceny), pro hokej i +/− po sezonách s
  poznámkou, že +/− závisí na spoluhráčích víc než na hráči.

### 2.3 Formace (fotbal) / Pětky (hokej)

Fotbal:

- Nejčastější základní jedenáctka v posledních 10 zápasech ve formaci,
  frekvence každého hráče v základu.
- Dvojice/trojice, které spolu hrály nejvíc minut (stoperská dvojice,
  střední záloha), s xG rozdílem týmu, když byly na hřišti spolu, vs. bez
  nich.

Hokej:

Pozice jsou **C (centr), LK (levé křídlo), PK (pravé křídlo), O (obránce),
B (brankář)**, plus hůl (levá/pravá). Centr má navíc úspěšnost na buly,
obránce čas v přesilovce a oslabení. Kádr i pětky používají tyhle pozice,
ne obecné „útočník“.

- **Pětky**: lajny útočníků (1.–4.) jako trojice **LK – C – PK** a obranné
  dvojice (1.–3.) tak, jak nastoupily v posledním zápase, s možností
  přepnout na „nejčastější v posledních 10“. U lajny: společný čas na
  ledě, GF/GA při 5 na 5, xGF% nebo CF% (kde data jsou), góly a nahrávky
  členů, buly centra. U obranné dvojice totéž plus levá/pravá strana.
- **Sehrané dvojice** (nezávisle na lajnách): dvojice hráčů, které spolu
  odehrály nejvíc času za posledních 2–3 sezony, i když dnes hrají v jiných
  lajnách nebo jeden přešel z křídla na centra. Trenéři dvojice přeskládávají,
  ale sehranost zůstává; ukazujeme ji jako síť (hráč → jeho nejčastější
  partneři, tloušťka čáry = společný čas).
- **WOWY** (with or without you) pro každou dvojici A–B, vždy ve třech
  stavech a vždy s počtem minut:

  | Stav | Co měříme | Čte se jako |
  |---|---|---|
  | **spolu na ledě** | xGF%, GF/60, GA/60 týmu, body A a B na 60 min | jak jim to jde společně |
  | **odděleně** (oba v sestavě, ale ne spolu) | totéž pro A bez B a pro B bez A | kdo z nich táhne dvojici |
  | **partner mimo sestavu** (zraněn, trest, zdravý náhradník) | totéž pro A v zápasech, kde B vůbec nehrál, a výsledky týmu v těch zápasech | co se stane, až B vypadne |

  Třetí stav je to, co uživatele i model zajímá nejvíc, a má typicky
  **nejmenší vzorek** (pár zápasů za sezonu). Proto: minimální práh
  minut, pod ním se zobrazí „málo dat“, a hodnoty se smršťují k týmovému
  průměru (empirický Bayes), ne surová čísla. Graf: tři sloupce xGF% se
  svislým intervalem nejistoty; čím kratší, tím víc dat.
- **Dopad absence přes dvojice**: na detailu zápasu („Kdo chybí“) ukážeme
  nejen `w[B] − w[náhradník]`, ale i **koho to nejvíc poškodí**: hráče A
  s největším regularizovaným rozdílem „spolu vs. bez B“. Text: „Bez B hraje
  A za posledních 3 sezony 41 minut 5 na 5, xGF% 44 % (spolu 56 %), málo dat“.
- **Speciální formace**: přesilovková a oslabovací jednotka 1 a 2,
  úspěšnost PP a PK týmu v sezoně a v posledních 10.
- **Brankářská dvojice**: jednička a dvojka, zápasy, % úspěšnosti zásahů,
  GAA, GSAx (kde je), rozpis startů v posledních 15 zápasech, kdo chytá
  dnes (potvrzeno / očekáváno). Graf % zásahů po sezonách pro oba.
- Vazba na model: když chybí hráč z 1. lajny, model ukáže, kdo se posune
  nahoru a kolik to stojí. Dopad absence B se počítá jako
  `w[B] − w[náhradník] + Σ_A chem[A,B]`, kde `chem[A,B]` je regularizovaný
  rozdíl výkonu A spolu s B vs. bez B (stav 2 a 3), smrštěný k nule podle
  počtu minut. Jestli ablace ve fázi 2 ukáže, že `chem` log‑loss nezlepšuje,
  zůstává jen na obrazovce a z modelu jde ven.

### 2.4 Trenéři a vedení

- Hlavní trenér: od kdy, bilance (Z/V/R/P, bodů na zápas), vlna týmu před
  a po nástupu (jediný graf, kde se ukazuje vlna přes víc než 10 zápasů,
  protože je to srovnání dvou období).
- Historie angažmá: klub, roky, bilance, úspěchy. Asistenti, trenér
  brankářů (hokej), sportovní ředitel / manažer.
- Historie trenérů klubu: časová osa, kdo kdy, s bilancí. Pro fanouška
  důležité, pro model jen datum poslední změny.

### 2.5 Historie a úspěchy

- Graf výkonu po sezonách: body na zápas, umístění, GF a GA na zápas,
  xGD na zápas (od roku, kdy je xG). Jeden graf, přepínatelná metrika.
- Trofeje a úspěchy: tituly, poháry, play‑off, s rokem.
- Rekordy klubu: nejvíc bodů, gólů, nejlepší střelec sezony.

### 2.6 Stadiony a cesty

- **Doma / venku**: bilance v sezoně a za posledních 5 sezon, GF/GA,
  bodů na zápas, rozdíl proti ligovému průměru domácí výhody.
- **Stadiony soupeřů**: tabulka arén, kde tým hrál venku, s bilancí za
  posledních N let, posledním výsledkem, kapacitou, vzdáleností od domova.
  **Vždy s počtem zápasů v závorce**, aby bylo vidět, že „0–3 v Třinci“ je
  tři zápasy. Žádná barva „prokleté místo“.
- **Venkovní tour** (hokej, hlavně NHL): sekvence venkovních zápasů v řadě,
  dny mezi zápasy, kilometry, časové zóny; bilance týmu na tourech délky 2,
  3, 4+. Ve fotbale totéž pro „anglický týden“ (3 zápasy v 7 dnech).
- Domácí stadion: kapacita, průměrná návštěva, bilance doma po sezonách.

## 3. Datový model (přírůstek ke kanonickému modelu z plánu)

```
Team(id, sport, name, league_id, venue_id)
Season(id, league_id, year_start)
Player(id, name, birth_date, nationality, position, shoots/foot)   # hokej position: C|LW|RW|D|G
PlayerContract(player_id, team_id, from, to, market_value)        # fotbal
PlayerStatus(player_id, as_of, status, reason, expected_return)
PlayerSeasonStats(player_id, team_id, season_id, gp, starts, minutes|toi,
                  goals, assists, plus_minus, xg, xa, rating, sv_pct, gaa, gsax)
PlayerMatchStats(player_id, match_id, minutes|toi, goals, assists, +/-, shots, rating)
Coach(id, name, birth_date, nationality)
CoachTenure(coach_id, team_id, role, from, to)                     # role: head, assistant, goalie, GM
Honour(subject_type, subject_id, kind, competition, season_id)     # subject: team|player|coach
Unit(id, team_id, kind, members[])      # kind: fwd_line, d_pair, five, pp1, pp2, pk1, pk2, goalie_tandem, cb_pair, mid3
UnitMatch(unit_id, match_id, toi_together, gf, ga, xgf, xga, cf, ca)
Duo(id, team_id, player_a, player_b)                                # sehrané dvojice napříč lajnami
DuoMatch(duo_id, match_id, state, toi, gf, ga, xgf, xga, cf, ca, pts_a, pts_b)
                                        # state: together | apart | b_absent | a_absent
PlayerAbsenceGame(player_id, match_id, reason)                      # kdo v zápase nehrál a proč
Venue(id, name, city, lat, lon, capacity, home_team_id)
TeamVenueRecord(team_id, venue_id, from_season, to_season, gp, w, d, l, gf, ga)   # materializované
RoadTrip(team_id, season_id, start_match_id, end_match_id, games, days, km, tz_changes, points)
```

`TeamVenueRecord` a `RoadTrip` jsou odvozené tabulky, přepočítávané
po každém zápase. `UnitMatch` je zdroj pro „pětky“. `DuoMatch` je zdroj pro
WOWY a pro `chem[A,B]`; stav `together` a `apart` vyžaduje střídání po
sekundách (shift charty), stav `*_absent` stačí ze sestav a `PlayerAbsenceGame`.

## 4. Zdroje dat

| Potřeba | Fotbal | Hokej NHL | Hokej Extraliga | Poznámka |
|---|---|---|---|---|
| Kádr, smlouvy, hodnoty, historie hráče | Transfermarkt (Apify scraper, nebo ručně), Sportmonks squads | NHL API (`api-web.nhle.com`, zdarma, oficiální), CapWages | hokej.cz / eliteprospects (ověřit podmínky) | |
| Statistiky hráčů po sezonách | Sportmonks, FBref (Apify), Understat (xG top ligy) | NHL API (G, A, +/−, TOI, střely), MoneyPuck / Natural Stat Trick (xG, GSAx, zdarma, CSV) | hokej.cz oficiální statistiky (G, A, +/−, TOI) | Extraliga nemá veřejné xG; CF/FF spočítáme ze střel, pokud poskytovatel dává pokusy |
| Sestavy, lajny, dvojice | Sportmonks lineups (formace, start/lavička), Sportradar widget (střídání s minutou) | NHL API shift charts → společný čas na ledě přesně; Daily Faceoff pro očekávané lajny a startujícího brankáře | Sportradar Global Ice Hockey lineups (ověřit, zda dává lajny) nebo zápisy z hokej.cz, kde jsou sestavy po pětkách | Lajny v Extralize jsou nejslabší místo; v MVP možná jen „z posledního zápisu“ |
| WOWY (spolu / odděleně / chybí) | obdoba: společné minuty z náhradních sestav a střídání (hrubé), stav „chybí“ ze sestav | NHL API shift charts (po sekundách) + MoneyPuck/Natural Stat Trick, které WOWY už publikují (zdarma, CSV) | **jen stav „chybí“** ze sestav a „spolu“ aproximovaný jako „ve stejné pětce v zápise“ (stejná lajna = celý zápas spolu). Bez shift chartů nelze „odděleně“ | Extraliga tedy dostane 2 ze 3 stavů a hrubě; UI to řekne („odhad ze sestav“) |
| Trenéři, historie angažmá, úspěchy | Transfermarkt (trenérské profily), Wikipedia/Wikidata (trofeje) | NHL API + Hockey‑Reference, Wikidata | hokej.cz, Wikidata | Wikidata jako jednotný zdroj trofejí a časových os; licence CC0 |
| Stadiony, souřadnice, kapacity | Wikidata, Sportmonks venues | NHL API venues, Wikidata | Wikidata | Vzdálenosti a časové zóny spočítáme sami (haversine je v Been Here, `lib/core/geo`) |
| Zranění, tresty | Sportmonks injuries, Flashscore (Apify) | NHL API injuries, Daily Faceoff | ruční kurace + hokej.cz | |

Rozpočet: NHL je prakticky zdarma a kompletní, to z něj dělá nejlepší ligu
pro vývoj této obrazovky. Fotbal top 5 je pokrytý Sportmonks + Understat.
**Chance Liga a Extraliga budou mít díry** (xG, lajny, GSAx) a UI s tím
musí počítat: prázdná buňka, ne nula.

## 5. Grafy

Vše jedním vizuálním systémem (viz `dataviz` skill při implementaci):

1. **Výkon po sezonách** (tým): čárový graf, osa X sezony, přepínatelná
   metrika (body/zápas, GF, GA, xGD, umístění obráceně). Svislé značky
   změn trenéra.
2. **Kariéra hráče**: sloupce góly + nahrávky po sezonách, čára minut/TOI,
   barevně klub. +/− zvlášť a s poznámkou.
3. **WOWY dvojice**: tři sloupce xGF% (spolu / odděleně / partner chybí)
   s intervalem nejistoty a počtem minut pod každým. Síť sehraných dvojic
   jako graf uzlů, tloušťka hrany = společný čas za 3 sezony.
4. **Brankáři**: % zásahů po sezonách pro jedničku i dvojku v jednom
   grafu, rozpis startů jako pásek 15 políček.
5. **Doma/venku**: dvě řady bodů na zápas po sezonách, ligový průměr
   domácí výhody jako šedé pásmo.
6. **Vlna před a po trenérovi**: jediný graf s vlnou přes delší období.
7. **Tour**: časová osa venkovních zápasů v řadě s body a kilometry.

Žádný graf nemá osu Y začínající jinde než v nule, kromě % zásahů
brankářů (0,850–0,950), kde to je řečeno v popisku.

## 6. Oblíbený tým a notifikace

- Uživatel si vybere 1–3 oblíbené týmy při onboardingu. Oblíbené týmy
  mají záložku na hlavní obrazovce a jsou vždy první v seznamu zápasů.
- Notifikace **jen** informační a vypnutelné po typu: potvrzená sestava /
  startující brankář (60 min před), změna statusu klíčového hráče, změna
  trenéra. **Nikdy** notifikace s kurzem nebo s doporučením sázky; to je
  hranice z `01_PLAN_PRO_OPUS.md`, §1 „Co appka nikdy nedělá“.

## 7. Zařazení do fází

| Co | Fáze | Důvod |
|---|---|---|
| Datový model (§3) a ingest kádrů, statistik hráčů, trenérů pro NHL a top 5 | **2** (spolu s absencemi) | Model absencí ty data potřebuje tak jako tak. |
| Společný čas dvojic/lajn (`UnitMatch`) pro NHL z shift chartů | **2** | Je to vstup pro `w[náhradník]`; ablace rozhodne, zda zůstane. |
| WOWY a `chem[A,B]` (`DuoMatch`) pro NHL; stav „chybí“ i pro Extraligu | **2** | Člen dopadu absence; ablace rozhodne. Na obrazovce zůstane i když z modelu vypadne. |
| Obrazovka Přehled + Kádr + Trenér (zjednodušená) | **5** (MVP) | Bez ní je appka jen seznam zápasů. |
| Pětky, brankářské dvojice, Historie, Stadiony, grafy | **6** | Retence. Vedle deníku to je důvod otevřít appku bez zápasu. |
| Extraliga lajny, Chance Liga xG | **8** nebo nikdy | Závisí na datech, která dnes nejsou. |

## 8. Co se nesmí stát

- „Úspěšnost na stadionu“ se nikdy neobjeví v odhadu zápasu ani v textu
  typu „tým v Třinci prohrál 5× v řadě“ bez počtu zápasů a období.
- Grafy po sezonách se neukazují na obrazovce zápasu. Patří k týmu.
- +/− hráče se neukazuje bez poznámky o jeho šumovosti a nikdy se podle
  něj neřadí.
- WOWY číslo se nikdy neukáže bez počtu minut a bez intervalu; pod prahem
  (výchozí 100 minut 5 na 5 za stav) je místo čísla „málo dat“. Věta „bez X
  tým prohrává“ bez vzorku je zakázaný výstup stejně jako „75 % úspěšnost“
  v plánu.
- Žádný řebříček „nejlepší hráč“ nad rámec statistik poskytovatele;
  vlastní rating hráče vymýšlet nebudeme, dokud model neprojde G1.
