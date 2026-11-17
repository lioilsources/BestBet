# Plán 05: MLS jako třetí pilot, fotbal (zadání pro Opus)

Stav 2026-11-17. Navazuje na plány 02–04. NHL a Allsvenskan běží v provozu (appka, kurzy, hinty, sestavy NHL). MLS je první fotbalová liga v repu, takže plán začíná rozdíly proti hokeji a teprve pak staví.

## 1. Rozhodnutí vlastníka (platí, neptat se znovu)

- MLS je třetí pilot po NHL a Allsvenskan. Vlastník zápasy sleduje (MLS vysílá Apple TV), karta zápasu má sloužit i jako příprava ke sledování.
- Témata, která vlastník výslovně chce:
  - formace (4-4-2 a ostatní): jak často je trenéři mění, proč a s jakým úspěchem;
  - držení míče, góly a handicapy po poločasech;
  - žluté a červené karty, fauly, rohy, penalty;
  - exekutoři penalt a při jakém skóre dali a nedali.
- Appka zůstává hint: nesází, nepřijímá peníze, žádný affiliate. Rozpočet 0 Kč.
- Merge jen na slovo vlastníka. Každá fáze = jeden PR.

Předpoklady plánu (nejsou rozhodnutí vlastníka; když je vlastník změní, platí jeho slovo):

- Stejné repo, fotbal jako druhý sport vedle hokeje. Hokejový kód se nepřepisuje, jen se z něj vyjme to, co je na sportu nezávislé.
- Trh je prior i tady (plán 02). Každý výsledek se měří proti zavírací ceně Pinnaclu, ne proti počtu bodů.
- Pravidlo `market-value-v1` se pro 1X2 MLS použije beze změny, v deníku a v Evaluaci zvlášť per liga. Cokoli nad to (handicap, rohy, karty) je nová verze pravidla až po bráně.

## 2. Kde to stojí

**Co je v repu hokejové a fotbal to rozbije:**

- `canon.Match`: `Periods` má tři třetiny, `Played()` to vyžaduje. Fotbal má dva poločasy.
- `games`: `home_60/away_60`, `ot_*`, `so_winner`, `ot_type` s hokejovými hodnotami.
- `internal/model/markets`: `ml`, `puck_line`, totaly 3,5–7,5, první třetina. Fotbal potřebuje asijský handicap se čtvrtinovými liniemi, totaly kolem 2,5 a trhy na poločas.
- `internal/oddspapi`: natvrdo `SportHockey`. `cmd/ingest-hockey`, `internal/hockey`, `internal/nhl`, `internal/swehockey`.
- `player_match_stats`: `toi_seconds`, `plus_minus`, `pim`.

**Co je na sportu nezávislé a použije se rovnou:** `devig`, `hint` (pravidlo nad 1X2), `timing`, `marketprior`, `httpcache`, `teammatch`, `fixture_links`, `odds_snapshots` (append-only), `lineups`, `coaches` a `coach_tenures`, API a web. Sloupec `leagues.sport` už existuje.

**Kalendář MLS (ověřit v S0):**

- Sezona 2026 je v play-off, finále je v prosinci. Živých zápasů letos zbývá jen několik.
- 2027 je přechodná: únor až květen, 14 zápasů na tým jen uvnitř konference, play-off na jeden zápas.
- Od července 2027 se hraje podzim–jaro se zimní pauzou. Píše se i o jedné tabulce místo konferencí.
- Důsledek: teď se staví na historii 2012–2026, živý provoz začíná v únoru 2027. Sondy na živých zápasech (S0) mají jen pár pokusů; co se nestihne, počká na únor.
- Klíč sezony: přechodná sezona 2027 i sezona 2027/28 začínají v roce 2027, takže „sezona = rok začátku“ koliduje. Návrh: u MLS je sezona rok, ve kterém končí (2012–2026 beze změny, přechodná 2027, pak 2028). Rozhodnout v S1 a zapsat do DECISIONS a `docs/DATA_CONTRACT.md`.

## 3. Rozdíly proti hokeji

Čísla v této tabulce jsou orientační z obecné znalosti. Skutečné hodnoty pro MLS změří S4, do té doby se necitují.

| Téma | Hokej (NHL) | Fotbal (MLS) | Co z toho plyne |
|---|---|---|---|
| Góly | kolem 6 na zápas | kolem 3 na zápas | Jeden gól váží víc. Nízká skóre potřebují opravu závislosti (Dixon–Coles), ne nezávislé Poissony. |
| Remíza | po 60 min asi čtvrtina, pak OT a nájezdy | zhruba čtvrtina zápasů a platí | 1X2 je hlavní trh, `ml` v základní části neexistuje. Prodloužení a penalty jen v play-off (formát ověřit). |
| Čas | 3 × 20 min čistého času | 2 × 45 min a nastavení | Gólů přibývá s časem, druhý poločas je bohatší. Model po poločasech, ne po třetinách. |
| Trhy | puck line ±1,5, total 5,5 | asijský handicap po čtvrtinách, total 2,5, poločasy, HT/FT, rohy, karty | Vypořádání sázky může být půl výhra nebo půl vrácení. `markets` i výpočet CLV to musí umět. |
| Domácí prostředí | malé | velké: cesty přes časová pásma, nadmořská výška, umělá tráva, horko | Výhoda domácích per stadion, ne jedna konstanta. |
| Sestava | známá medián 13,5 min před začátkem (B0 neprošla) | základní jedenáctka zhruba hodinu před výkopem (ověřit) | Analogická brána má šanci projít. Trh má čas reagovat, CZ knihy možná pomaleji. |
| Střídání | letmé, čtyři lajny, TOI | 5 střídání, 11 hráčů hraje většinu zápasu | Místo TOI minuty a starty. „Pravidelný hráč“ se definuje přes starty. |
| Vyloučení | 2 min, vrací se | červená karta do konce zápasu, stop na další zápas | Zápasy s červenou zkreslují držení, rohy i góly. Stopky jsou absence známé předem. |
| Rozhodčí | čtyři, malý vliv na trhy | jeden hlavní a VAR, velký vliv na karty, fauly a penalty | Rozhodčí je vstup pro karty a penalty. V hokeji takový vstup není. |
| Penalta | trestné střílení je vzácné | zhruba jedna na tři až čtyři zápasy, proměňuje se asi tři čtvrtiny | Má smysl evidence exekutorů. Malé vzorky, proto shrinkage. |
| Tlak a držení | střely, Corsi | držení míče, xG | Tým, který prohrává, drží míč víc a kope víc rohů. Surové držení bez opravy na stav zápasu není síla týmu. |
| Taktika | lajny, přesilovky | formace, trenér ji mění mezi zápasy i během nich | Nový druh vstupu. Formaci přiřazuje zdroj dat, je to cizí interpretace. |
| Brankář | největší jednotlivý vliv, vlastní vrstva | součást jedenáctky | Okno brankáře z plánu 02 se nepřenáší. |
| Kádr | 23 hráčů, strop | platový strop a pár hvězd mimo něj (Designated Players), reprezentační pauzy, poháry | Absence hvězdy váží víc. Reprezentační absence a rotace kvůli pohárům jsou známé předem. |
| Vzorek | 1 312 zápasů za sezonu | asi 510 zápasů za sezonu, noví účastníci ligy | Širší intervaly, potřeba víc sezon. Od 2012 je to řádově 6 000 zápasů. |

## 4. Zdroje dat (vše zdarma)

| Zdroj | Co dává | Stav 2026-11-17 |
|---|---|---|
| football-data.co.uk, soubor `new/USA.csv` | výsledky MLS od 2012 a zavírací kurzy 1X2: Pinnacle, maximum, průměr, Betfair Exchange, Bet365 | Hlavička a první řádky ověřeny. Bez poločasů a statistik. |
| American Soccer Analysis, veřejné API (`/api/v1/mls/…`) | `games`, `games/xgoals`, `games/shots`, `games/periods`, `games/game-flow`, `managers`, `referees`, `stadia`, hráči a týmy (xG, g+, platy) | Seznam cest ověřen ze specifikace, klíč v ní není. Tvar odpovědí `shots`, `periods` a `game-flow` neověřen. Sestavy, formace a karty ve specifikaci vidět nejsou. |
| API-FOOTBALL (stejný poskytovatel jako API-HOCKEY) | sestavy s formací a trenérem, události (góly, karty, střídání, penalty, VAR), statistiky zápasu (držení, fauly, rohy, střely) | Neověřeno. Free tier má 100 dotazů denně a jen několik starších sezon; rozdělení statistik po poločasech neověřeno. Potřebuje klíč od vlastníka. |
| Web ligy (zápasové centrum mlssoccer.com) | kandidát na potvrzenou sestavu a formaci před výkopem | Neověřeno, ani existence veřejného JSON, ani podmínky užití. Stejný přístup jako u veřejného NHL API. |
| OddsPapi | živé kurzy a historie | Neověřeno pro fotbal: ID turnaje MLS, které trhy (handicap, poločasy, rohy, karty) a které CZ knihy. Historie ve free tieru sahá jen od ledna 2026. |
| FBref | dřív pokročilé statistiky | Nestavět na něm. V lednu 2026 mu dodavatel dat vypověděl smlouvu a pokročilé statistiky zmizely. Žádný scraping. |

Apple TV je jen na dívání, žádná data se z něj neberou.

Rozpočet dotazů API-FOOTBALL: zápas stojí 3 dotazy (události, sestavy, statistiky). Sezona o 510 zápasech je asi 1 530 dotazů, tedy 16 dní při 100 denně. Zpětné plnění proto běží jako pomalý cron, ne jako jednorázová dávka.

## 5. Fáze

### S0: ověření zdrojů (bez PR, výsledek do DECISIONS)

1. Pro každé téma z §1 najít zdroj a zapsat tabulku téma × zdroj × sezony × počet dotazů: formace a sestavy, události s minutou (góly, karty, střídání, penalty), statistiky týmů (držení, fauly, rohy), totéž po poločasech.
2. ASA: stáhnout jeden zápas z `games/shots`, `games/periods` a `games/game-flow` a popsat pole. Zvlášť: je u střely z penalty střelec, minuta, výsledek a skóre před ní?
3. API-FOOTBALL (až bude klíč): které sezony MLS free tier pustí, zda `fixtures/statistics` umí poločasy, jak vypadá formace a mřížka v `fixtures/lineups`. Nejvýš 20 dotazů.
4. OddsPapi: turnaj MLS, seznam trhů a knih na jednom zápase play-off.
5. Sonda sestav na jednom zápase play-off: od −120 min do výkopu po 10 min. Kdy se poprvé objeví jedenáctka a formace?
6. Ověřit pravidla, o která se plán opírá: formát play-off, hranice žlutých karet pro stop, reprezentační pauzy, datum finále, kalendář 2027.

- **Brána Z0a:** každé téma z §1 má bezplatný zdroj aspoň pro tři sezony. Téma bez zdroje se nahlásí vlastníkovi s cenou placené varianty; nerozhodovat.
- **Brána Z0b:** jedenáctka je k dispozici ≥ 45 min před výkopem. Když ne, sestava v appce zůstane odhadem z posledního zápasu, jako u Allsvenskan.

### S1: fotbal v datech (PR)

- `config/leagues.yaml`: liga `usa-mls`, `sport: soccer`, klíč sezony podle §2. `leagues.sport = 'soccer'`.
- `internal/soccer` (čistá logika, testy bez sítě): výsledek po 90 min, poločasy, prodloužení a penalty, vypořádání asijského handicapu a totalu včetně čtvrtinových linií.
- `canon.Match`: počet částí podle sportu (`Played()` nesmí předpokládat tři). Hokejové testy zůstávají zelené.
- `games` bez nových sloupců: `home_60/away_60` znamená skóre po základní hrací době (u fotbalu 90 min a nastavení), `period_scores` drží poločasy, `decided_in` `ot` = prodloužení, `so` = penalty. Nové hodnoty `ot_type` novou migrací (nasazené migrace se needitují). Mapování zapsat do `docs/DATA_CONTRACT.md`.
- `cmd/ingest-soccer` nad stejným `internal/store`:
  - `fdcouk`: zápasy 2012–2026 a zavírací kurzy do `odds_snapshots` (zdroj `fdcouk`, `closing`);
  - `asa`: ID zápasu, xG, stadion, rozhodčí, trenér jako sekundární zdroj (doplňuje prázdná pole, nic nepřepisuje).
- Názvy týmů přes `teammatch` a `team_aliases`; přejmenované a nové kluby ručně do configu.
- `internal/report`: sekce MLS v `DATA_M0.md`.
- **Brána Z1:** ≥ 99 % zápasů spárovaných mezi oběma zdroji, zavírací kurz Pinnaclu u ≥ 95 % zápasů. Jako laťka se rovnou spočítá log-loss zavíracího trhu po sezonách; každý pozdější model stojí vedle ní.

### S2: kurzy a hinty MLS (PR, jen když S0 potvrdí OddsPapi)

- `internal/oddspapi`: sport jako parametr, ne konstanta.
- `ingest-soccer oddspapi -league usa-mls -live`, hinty pravidlem `market-value-v1` nad 1X2. Deník i Evaluace per liga.
- Appka: přepínač ligy, MLS na kartách Dnes, Zápas, Evaluace, Data.
- Načasování jako L6 (pohyb Pinnaclu ≥ 2 p. b. za ≤ 30 min, reakce knihy ≥ 1 p. b.), až bude historie aspoň 100 zápasů. Z něj vyjde perioda cronu.

### S3: události, statistiky, formace (PR)

Nové tabulky jednou migrací:

- `match_events`: zápas, minuta, nastavení, poločas, tým, hráč, typ (`goal`, `own_goal`, `penalty_scored`, `penalty_missed`, `penalty_saved`, `yellow`, `second_yellow`, `red`, `sub_in`, `sub_out`, `var`), skóre před událostí, zdroj.
- `match_team_stats`: zápas, tým, rozsah (`full`, `h1`, `h2`), držení, střely, střely na bránu, fauly, rohy, ofsajdy, žluté, červené, xG, zdroj.
- `match_formations`: zápas, tým, formace jak ji uvádí zdroj, rodina (S5), trenér, zdroj.
- `referees` a vazba na zápas.
- Sestava do existující `lineups`, minuty hráčů do `player_match_stats` (`toi_seconds` = minuty × 60, hokejová pole NULL).

Zpětné plnění cronem s denním rozpočtem, nejnovější dostupná sezona první. Penalty z penaltového rozstřelu se ukládají odděleně a do statistik penalt se nepočítají.

- **Brána Z3:** pokrytí ≥ 95 % zápasů dostupných sezon; součet gólů z událostí sedí se skóre, počty karet z událostí se statistikami. Rozdíly se vypíšou, neopravují se ručně.

### S4: popis ligy (PR, report `docs/RESULTS_M4_MLS_POPIS.md`)

Jen popis, žádná brána. Každé číslo s n a intervalem; kde se testuje, BH přes celou rodinu testů v reportu. Zápasy s červenou kartou vždy zvlášť i dohromady.

- **Poločasy:**
  - podíl gólů v 1. a 2. poločase, po patnáctiminutovkách a v nastavení;
  - tabulka HT/FT (9 polí) a jak často se vedení v poločase udrží;
  - handicapy po poločasech: jak často by favorit pokryl linie 0, −0,25, −0,5, −0,75 a −1 v 1. poločase, ve 2. poločase samotném a v celém zápase, rozděleno do třetin podle síly favorita ze zavíracího trhu;
  - kontrola: vysvětlí poločasy model, který zná jen zavírací 1X2 a total a dělí góly mezi poločasy konstantou? Když OddsPapi má trhy na poločas, vedle toho trh.
- **Držení míče:** rozdělení, doma a venku, po poločasech. Efekt stavu: změna držení ve 2. poločase podle skóre v poločase. **H8:** průměrné držení z posledních 5 zápasů nepředpovídá výsledek nad rámec zavíracího trhu (čeká se nula; hlásit β a interval).
- **Karty:** žluté a červené na zápas, doma a venku, po čtvrthodinách, podle stavu skóre. Rozhodčí: karty na zápas se shrinkage k průměru ligy a s počtem zápasů. Efekt červené: poměr gólových intenzit týmu v deseti a soupeře po kartě proti stavu před ní.
- **Fauly:** na zápas, poměr faulů na kartu per rozhodčí, doma a venku.
- **Rohy:** na zápas a po poločasech, vztah ke střelám a xG, efekt stavu (prohrávající tým), rozptyl proti průměru (rozhodne mezi Poissonem a negativně binomickým rozdělením v S7).
- **Penalty:** četnost na zápas, doma a venku, per rozhodčí se shrinkage, před VAR a s ním.

### S5: formace (PR, report `docs/RESULTS_M4_MLS_FORMACE.md`)

Definice jsou pevné předem a po pohledu na výsledky se nemění. Zapsat do DECISIONS před prvním během.

- **Rodina formace:** obrana podle prvního čísla (3, 4, 5), útok podle posledního (1, 2, 3). Vedle rodiny se drží přesný zápis zdroje (4-4-2, 4-2-3-1, 4-3-3, 3-5-2, 3-4-3, 5-3-2 a další).
- **Změna formace:** výchozí rodina se liší od předchozího ligového zápasu týmu pod stejným trenérem. První zápasy nového trenéra jsou zvláštní kategorie.
- **Změna během zápasu:** z výchozích sestav zjistit nejde. Jen když zdroj dává pozice střídajících, počítá se náhrada obránce útočníkem (a naopak) podle stavu skóre. Jinak se v reportu napíše „z dat nezjistitelné“; nic se nedomýšlí.

**Jak často:**

- podíl zápasů per formace a sezona (kolik z toho 4-4-2), trend od 2012;
- per trenér: počet různých formací za sezonu, podíl zápasů se změnou, pravděpodobnost, že zůstane u té minulé.

**Proč** (jen příznaky, které jdou změřit; je to souvislost, ne trenérův motiv):

- výsledek minulého zápasu; série ≥ 3 zápasů bez výhry;
- body proti očekávání trhu za posledních 5 zápasů;
- chybějící pravidelní hráči podle řady (obrana, záloha, útok), včetně stopek za karty;
- soupeř: jeho obvyklá rodina a zda je podle trhu favorit;
- doma nebo venku; ≤ 4 dny od minulého zápasu; prvních 5 zápasů nového trenéra.

Logistická regrese změny na tyto příznaky, hlásit poměry šancí s intervaly.

**S jakým úspěchem:**

- Měřítko je odchylka od trhu: body minus očekávané body ze zavíracích pravděpodobností Pinnaclu (Shin). Surové body ne: tým mění formaci, když se mu nedaří, a návrat k průměru by vypadal jako úspěch změny.
- Srovnání zápasů se změnou a bez ní uvnitř stejného týmu a sezony, a zvlášť podle důvodu (vynucená absencí proti dobrovolné).
- **H9:** změna formace nepřináší výsledek nad zavírací trh (čeká se nula).
- **H10:** pohyb trhu mezi posledním snímkem před zveřejněním sestav a zavírací cenou souvisí se změnou formace. Říká, jestli trh formaci vůbec čte. Jen na zápasech s oběma snímky.
- Tabulka rodina proti rodině (např. tři obránci proti čtyřem): průzkumná, BH, bez nároku na závěr.
- **Brána Z5:** vrstva `formation_change` jde do modelu, jen když Δ log-loss proti trhu jako prioru < 0 s intervalem pod nulou na odložených sezonách a projde BH. Jinak je formace jen informace v appce.

### S6: penalty a exekutoři (PR, report `docs/RESULTS_M4_MLS_PENALTY.md`)

- **Záznam penalty:** zápas, minuta, poločas, tým, exekutor, brankář, výsledek (`scored`, `saved`, `missed`), skóre před penaltou z pohledu exekutora (−2 a horší, −1, 0, +1, +2 a lepší), doma nebo venku, fáze soutěže. Gól z dorážky není proměněná penalta.
- **Liga:** úspěšnost podle stavu skóre, podle času (0–45, 46–75, 76+) a doma a venku, Wilsonovy intervaly, test shody. Do reportu napsat i sílu testu: rozdíl 5 p. b. mezi dvěma stavy potřebuje řádově 1 100 penalt na skupinu, liga jich má od 2012 celkem řádově 2 000. Co se z toho usoudit nedá, se napíše výslovně.
- **Exekutor:** pokusy, proměněné, úspěšnost stažená k průměru ligy (beta-binomický prior z dřívějších sezon). Rozpis podle stavu skóre jako počty; procenta jen při n ≥ 10.
- **Hlavní exekutor týmu:** nejvíc pokusů za posledních 12 měsíců mezi hráči, kteří byli na hřišti. Sleduje se, kdy penaltu kopal někdo jiný, ač hlavní exekutor hrál.
- **Tým a rozhodčí:** penalty získané a zaviněné na zápas, se shrinkage.
- Do modelu nic. Použití je karta zápasu (S8). Sázky na střelce by byly nové pravidlo a nové kurzy, to je rozhodnutí vlastníka.

### S7: model (PR, až po S1 a S3)

- Baseline `halves-v1`: dvourozměrný Poisson s opravou Dixon–Coles, intenzity po poločasech, časové vážení, výhoda domácích per stadion, walk-forward jako hokejový `periods`. Hyperparametry pevně předem.
- Trhy z jedné matice skóre: `1x2`, `dc`, `dnb`, `total` a `ah` včetně čtvrtinových linií, `btts`, `h1_1x2`, `h1_total`, `h1_ah`, `htft`.
- Vrstvy nad trhem jako priorem (`marketprior.Walk`): forma podle xG, odpočinek a cesta, absence z jedenáctky (definice z plánu 04 převedené na starty a minuty, zapsat předem), změna formace (jen po Z5).
- Počty (rohy, karty): model s útokem a obranou týmu a s rozhodčím. Proti trhu se hodnotí, jen když OddsPapi ty trhy u CZ knih má; jinak zůstanou popisem.
- **Dělení sezon (návrh, zapsat před prvním během):** 2012–2014 jen zahřívací, ladění do 2023, odložené 2024–2026 a vše pozdější.
- **Brána Z7:** stejná jako A1 a B2: Δ log-loss proti trhu jako prioru < 0, interval pod nulou na odložených sezonách, BH. V hokeji A1 ani B2 neprošly; záporný výsledek se hlásí stejně jako kladný a pravidlo hintu se bez brány nemění.

### S8: karta zápasu MLS v appce (PR, může jít po částech za S3–S6)

Blok „Co sledovat“ u zápasu, vždy vedle čísel trhu (invariant):

- sestava a formace obou týmů se stavem (odhad, potvrzená), příznak změny formace proti minulému zápasu a jak často ji trenér mění;
- chybějící pravidelní hráči a stopky;
- rozhodčí: karty a penalty na zápas, se shrinkage a počtem zápasů;
- exekutor penalt a jeho bilance podle stavu skóre;
- profil poločasů obou týmů (góly, držení), průměry rohů a faulů;
- hint 1X2 jako dosud.

Karta Data: pokrytí fotbalových tabulek. Snímky obrazovky do PR.

### S9: provoz

- Cron pro MLS do `deploy/crontab.example`: kurzy (perioda podle S2), sonda sestav před výkopem (podle Z0b), pomalé zpětné plnění API-FOOTBALL.
- Deklarované akce na JODĚ jsou stejné jako v plánu 04 §4, navíc binárka `ingest-soccer`. Žádné sudo, nikdy `compose down`, každá akce do deníku nasazení.
- Kvóty OddsPapi a API-FOOTBALL sledovat; při 429 zpomalit a zapsat.
- Živý provoz naplno od února 2027. Do té doby běží jen zpětné plnění a to, co stihne play-off 2026.

## 6. Pravidla práce

- Před fází přečíst tento plán, `docs/DECISIONS.md` a `CLAUDE.md`. Po fázi `make check`, PR, zápis do DECISIONS.
- Kód a komentáře anglicky, dokumenty a PR česky. Veřejné repo: žádné klíče, LAN adresy ani domovské cesty.
- Testy bez sítě a bez `time.Now`; `model`, `features` a `soccer` neznají DB ani HTTP.
- Definice a prahy z S4–S7 se zapisují před prvním během a po pohledu na výsledky se nemění. Odložené sezony se k ladění nepoužívají.
- Metrika modelu nikdy bez trhu vedle. U malých vzorků počty místo procent.
- Žádný scraping webů, které ho zakazují, a žádná data ze streamu. Každý nový zdroj přes `httpcache` s rozpočtem.
- Po bráně se nezastavovat, hlásit krátce: co prošlo, čísla, co dál.

## 7. Kdy se ptát vlastníka

- Klíč API-FOOTBALL: registrace je zdarma, klíč zadává vlastník do `.env` (`APIFOOTBALL_KEY`). Bez něj S3 stojí na ASA a webu ligy.
- Z0a: téma nemá bezplatný zdroj. Předložit varianty s cenou.
- Klíč sezony MLS, pokud návrh z §2 naráží na existující kód víc, než se čekalo.
- Z5 nebo Z7 projde: návrh nové verze pravidla hintu, než vznikne první záznam.
- Nové trhy v hintech (handicap, rohy, karty, střelci).

## 8. Zadání do chatu

> Implementuj podle `Prompts/05-PLAN-M4-mls-fotbal.md`. Začni S0. Rozhodnutí v §1 platí, neptej se na ně znovu. Hlas po každé bráně.
