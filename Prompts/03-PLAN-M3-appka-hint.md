# Plán 03: rychlý běh do appky (hint pro sázkaře)

Stav 2026-11-08. Navazuje na plán 02, kde brána A1 neprošla (`docs/DECISIONS.md` 2026-11-08). Vlastník rozhodl, že appka vznikne hned jako **ukázka dat, práce s nimi a evaluace**. Appka nesází a nepřijímá sázky. Je to hint s přidanou hodnotou pro sázejícího a feedback vlastníka určí, co přidat a co přehodnotit.

## 1. Rozhodnutí vlastníka (2026-11-08)

- M3 se otevírá, i když G1 a A1 neprošly. Appka je informační nástroj: nesází, nepřijímá peníze, žádný affiliate.
- Přidaná hodnota není předpověď lepší než trh. Je to **informace z trhu**: rozdíl měkkých knih proti Pinnaclu, nejlepší cena napříč knihami, pohyb kurzu a forma týmů.
- Rozpočet zůstává 0 Kč.
- 2026-11-10: appka ukáže i sestavy a zranění z veřejného NHL API (§7).

## 2. Co appka ukáže

1. **Dnes:** zápasy v příštích 48 h. U každého:
   - férová pravděpodobnost Pinnaclu (Shin) a její pohyb od otevření,
   - náš model vedle trhu,
   - nejlepší CZ cena na každý výběr,
   - hint, pokud ho pravidlo `market-value-v1` dá.
2. **Zápas:** všechny nabídky po knihách, graf pohybu kurzu, vlna obou týmů a příspěvky modelu.
3. **Data:** co máme. Zápasy po ligách, kurzy po knihách a jejich čerstvost, brankáři, trenéři.
4. **Evaluace:**
   - papírový deník hintů: CLV proti uzavíracímu Pinnaclu s intervalem, yield, počet;
   - reporty backtestů (G1, F1), jak jsou v `docs/`.

Každá cena nese varování podle § 5j.

## 3. Pravidlo `market-value-v1` (pevné předem, prospektivní test)

Pravidlo je stejné jako sázka „trh sám“ z F1, kde bylo jen průzkumné. Tady se poprvé měří dopředu.

- **Vstup:** poslední kurzy před začátkem zápasu, trh 1X2.
  - p_sharp = Shin z celého 1X2 Pinnaclu.
  - p_soft = Shin z celého 1X2 knihy `betano.cz` nebo `merkurxtip.cz`.
- **Kdy je hint:**
  - hrana p_sharp − p_soft ≥ 3 p. b.,
  - EV = p_sharp × kurz − 1 > 0,
  - kurz ≤ 10.
  Ze všech výběrů a knih vyhrává nejvyšší EV.
- **Zápis do deníku:** první chvíle, kdy zápas pravidlo splní, s tou cenou. Na zápas jeden záznam (`hints`, verze pravidla). Později se nepřepisuje.
- **Vyhodnocení:**
  - CLV = ln(kurz × p_close), kde p_close je Shin uzavíracího 1X2 Pinnaclu;
  - výsledek po 60 minutách;
  - yield při flat vkladu.
- **Brána M2-hint:** CLV > 0 s 95% intervalem nad nulou na ≥ 300 záznamech s uzavíracím Pinnaclem. Do té doby appka hint ukazuje jako „ve zkoušce“.

## 4. Technika

- **Backend:** `cmd/api` rozšířený o `/api/*` a o statické UI přes `go:embed` (HTML + JS bez buildu). Vybráno kvůli rychlosti: jedna binárka, žádný Node. Flutter klient může přijít po feedbacku nad stejným API.
- **Živé kurzy:**
  - `ingest-hockey oddspapi` pro okno −30 h až +36 h, knihy `pinnacle,betano.cz,merkurxtip.cz`, cron po 2 h.
  - Cena, která ještě není konečná, se ukládá jako `adhoc`. Pozdější běh ji přeznačí na `t24h`, `t1h` nebo `closing`.
- **Deník:** `ingest-hockey hints` po každém stažení kurzů.
- **Nasazení:** služba `api` v `deploy/joda/compose.yml` (distroless, binárka z `bin/`), port 8096. Přístup jen z LAN a tailnetu.

## 5. Omezení

- Bez klíče `ODDSPAPI_KEY` v `.env` na JODĚ nejsou živé kurzy a hinty nevznikají. Klíč vloží Olin po rotaci.
- Free tarif OddsPapi má jen Betano a MerkurXtip z CZ knih. Kvóta je neznámá, proto cron po 2 h a jen 3 knihy.
- 300 záznamů je při ~1–2 hintech denně několik měsíců.

## 6. Co se nedělá

Žádné sázení ani přijímání sázek, žádný affiliate, žádný obchod (App Store, Play) a nic placeného.

## 7. Sestavy a zranění (rozhodnutí vlastníka 2026-11-10)

Vlastník potvrdil, že „informace, které trh skrývá“ jsou i informace mimo kurzy: sestavy a zranění. Rozpočet zůstává 0 Kč, zdroj je veřejné NHL API (jako u brankářů).

### 7.1 Co je zdarma a kdy

| Informace | Zdroj | Kdy je k dispozici |
|---|---|---|
| Kdo hrál, s jakým TOI, body, střelami | NHL `gamecenter/{id}/boxscore` po zápase (dnes z něj bereme jen brankáře) | po zápase; historie je v HTTP cache, takže zpětné doplnění 2024/25–2026/27 nestojí žádný request |
| Potvrzená sestava před zápasem | tentýž boxscore, jakmile zápas přejde do stavu `PRE` (rozbruslení); **neověřeno, viz plán 04 L0** | zhruba 30–60 min před začátkem, stejné okno jako brankář (H3) |
| Škrty (`scratches`) | `gamecenter/{id}/right-rail` | až po zápase, bez důvodu |
| Soupiska týmu | `roster/{tým}/current` | kdykoli, ale obsahuje i hráče na IR (28 jmen), zraněné neoznačuje |
| Důvod absence (zranění, trest, zdravý škrt) | NHL API **nemá** | nikdy; důvod je `unknown` |

Zpravodajství o zraněních (DailyFaceoff a podobné) je zdarma jen technicky. Je to scraping cizího webu s nejasnými podmínkami a do veřejného repa nejde bez rozhodnutí vlastníka. Výchozí: **ne**.

Allsvenskan: swehockey má sestavu v zápasovém protokolu po zápase. Zda ji zveřejňuje i před zápasem, ověří krok 5 v §7.5.

### 7.2 Co appka ukáže

- **Zápas, blok „Sestava“:** stav `odhad` (sestava z posledního zápasu týmu) nebo `potvrzeno` s časem potvrzení.
  - **Chybí:** pravidelní hráči, kteří v sestavě nejsou. Pravidelný = hrál ≥ 60 % z posledních 20 zápasů týmu a má TOI na zápas nad mediánem týmu. U každého TOI/zápas, body/zápas, počet vynechaných zápasů.
  - **Vrací se:** hráč pravidelný před absencí, který se vrací po ≥ 3 vynechaných zápasech.
  - **Váha absencí:** podíl TOI chybějících na TOI týmu (`absence_toi_share`) a podíl bodů. Jedno číslo na tým.
- **Dnes:** štítek „chybí N klíčových“ na kartě, kde klíčový = TOI/zápas v horní třetině týmu.
- Absence nevstupuje do hintu. Pravidlo `market-value-v1` se nemění, aby prospektivní test zůstal čistý. Případná verze s absencemi bude nová verze pravidla s vlastním deníkem.

### 7.3 Evaluace (pevně předem)

- **H7 (obdoba H3 pro brankáře):** vysvětlují absence pohyb Pinnaclu mezi t24h a uzavřením? Regrese Δlogit(p_close) − Δlogit(p_t24h) na rozdílu `absence_toi_share` hosté − domácí. Hlásí se β s intervalem a podíl vysvětleného rozptylu, zvlášť pro absence známé už z posledního zápasu (odhad) a pro nové (až z potvrzené sestavy).
- **F-abs:** vrstva `absence` v modelu, β·(share_away − share_home) na rozdíl síly, stejné pravidlo jako fáze 2 (Δ log-loss proti baseline, BH přes vrstvy). Jen NHL, Allsvenskan až bude mít data.
- **Potvrzená sestava proti trhu (až přijdou živé kurzy):** kolik hintů vzniklo před potvrzením sestavy a jaké mají CLV proti hintům po něm. Jen popis, bez brány.
- Do modelu a do hintu jde vrstva až po průchodu F-abs, do UI jde hned (00 §1.4.2: informace smí být narativ).

### 7.4 Technika

- Tabulky z fáze 0 už existují a jsou prázdné: `players`, `player_sources`, `player_match_stats`, `player_absence_games`. Žádný kód je zatím neplní.
- `ingest-hockey nhl`: z boxscore navíc uloží bruslaře do `player_match_stats` (TOI, G, A, +/−, střely, PIM, `line_no` z pořadí v boxscore) a hráče do `players` + `player_sources (nhl, playerId)`. Zpětně pro 2024/25–2026/27 z cache.
- `ingest-hockey nhl-lineups`: pro dnešní zápasy stahuje boxscore v okně −90 min až start každých 10 min. Jakmile je v něm sestava, uloží ji do nové tabulky `lineups` (game_id, team_id, player_id, confirmed_at, source; migrace 0006). Po zápase doplní `player_absence_games` ze škrtů a z rozdílu pravidelní − sestava, důvod `unknown`.
- „Chybí“ a `absence_toi_share` se počítají dotazem nad `player_match_stats` a `lineups`, bez další tabulky.
- API: `/api/games/{id}` přidá `lineup` {status, confirmed_at, missing[], returning[], share}; `/api/board` přidá `missing` na tým.
- Cron: `nhl-lineups` každých 10 min, bez zápasů v okně skončí bez requestu.
- Kvóta NHL API: žádný klíč, limit neznámý; dnes jedeme ~1 request/s. Lineups přidají v herní den ~15 × počet zápasů.

### 7.5 Pořadí

1. Bruslaři z boxscore a zpětné doplnění z cache (NHL 2024/25–2026/27).
2. Tabulka `lineups`, příkaz `nhl-lineups`, cron.
3. UI: blok Sestava, štítek na kartě.
4. Backtest H7 a F-abs na NHL (data za 2 sezony jsou hned).
5. Allsvenskan: ověřit sestavu před zápasem na swehockey; pak stejný mechanismus.

### 7.6 Omezení

- Důvod absence neznáme. Zranění, trest a zdravý škrt jsou jedno „chybí“.
- Potvrzená sestava přichází asi hodinu před zápasem. Měkké knihy do té doby kurz často stáhnou, takže informace z potvrzené sestavy může přijít pozdě. Přesně to měří H7.
- Odhad z posledního zápasu nerozliší jednorázový škrt od dlouhé absence; „vynecháno N zápasů“ to ukáže.
