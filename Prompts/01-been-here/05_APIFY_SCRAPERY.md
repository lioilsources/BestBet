# Apify Store a MCP — co tam je pro sázkovky

Ověřeno 2026-11-05 dotazy na veřejné API Apify Store (`api.apify.com/v2/store`)
a čtením stránek jednotlivých actorů. Čísla „users30d“ jsou měsíční
uživatelé actoru podle Apify.

## Odpověď na otázku jednou větou

Scraper na **Tipsport, Fortunu, Chance ani Sazkabet v Apify Store neexistuje**
(0 výsledků; hity na „fortuna“ jsou Fortune 500). Existuje scraper na
**Betano včetně Česka**, desítky scraperů na **Pinnacle**, **OddsPortal**,
**Flashscore** a **Sofascore**, a scraper **Sportradar widgetu**, který dává
sestavy, lavičku a střídání s minutou. Všechno je pay‑per‑event od jednotlivých
vývojářů s jednotkami až desítkami uživatelů měsíčně.

## Co jsem našel

### České sázkovky

| Hledáno | Actorů | Poznámka |
|---|---|---|
| tipsport, tipsport.cz | 0 | Výsledky jsou jen TheSportsDB wrappery, které se chytly na fulltext. |
| fortuna, ifortuna | 0 relevantních | Všechno Fortune 500 / Fortune magazín. |
| chance, sazkabet | 0 | |
| betano | 3 | `blackfalcondata/betano-odds-api` (11 users/měs), `blackfalcondata/betano-scraper` (10), `getascraper/betano-odds-scraper` (0). |

**Betano actor** (`blackfalcondata`): 13 zemí včetně **Česka**, 23 sportů
včetně hokeje a e‑sportu, předzápas i live, hendikepy a O/U potvrzené,
„600+ trhů na zápas“; poločasy, přesný výsledek a třetiny stránka
nepotvrzuje (vidět je jen kód trhu `H2HT`). Cena: $1,50 / 1 000 dotazů,
plné trhy $0,012 / zápas, denní pas $3. 39 uživatelů celkem, 99,8 %
úspěšných běhů. Disclaimer: uživatel odpovídá za dodržení podmínek Betana.

### Referenční kurzy (Pinnacle, burzy)

| Actor | users/měs | Co dává |
|---|---|---|
| `lergassy/pinnacle-odds-api` | 25 | Ostré kurzy, limity, fair odds. |
| `parseforge/pinnacle-odds-scraper` | 12 | Kurzy. |
| `sports-odds-lab/pinnacle-odds` | 1 | **Uzavírací kurzy**, no‑vig fair odds, limity. |
| `scrapemint/sports-odds-scraper` | 56 | DraftKings + Pinnacle + Bovada. |
| `lergassy/sports-odds-api` | 46 | Pinnacle + DraftKings + arbitráž. |
| `yakk000/betfair-exchange-spain-odds` | 2 | Betfair, ale jen španělská burza. |

Pinnacle zavřel API veřejnosti 23. 7. 2025; tyhle actory čtou veřejný
guest web. Jako **referenční cena pro backtest** je to použitelné. Jako
produkční závislost je to jedna změna frontendu od pádu.

### Agregátory kurzů (historie, uzavírací kurzy)

| Actor | users/měs | Co dává | Cena |
|---|---|---|---|
| `piotrv1001/oddsportal-scraper` | 13 (78 celkem) | **Otevírací i uzavírací kurzy s časem, 20+ sezon zpět**, všechny trhy (1X2, AH, EH, O/U, přesný výsledek, HT/FT, poločasy), všechny sporty včetně hokeje. Množina sázkovek závisí na proxy zemi; **česká sázkovka nepotvrzena**. OddsPortal `robots.txt` scraping zakazuje. | $3 / 1 000 zápasů; celá sezona s uzavíracími kurzy ≈ $1,14 |
| `neverempty/oddsportal-historical-odds-scraper` | 1 | Historie, otevírací kurzy. | |
| `sports-odds-lab/football-odds-results` | 18 | Flashscore: 1X2 od 2005/06, O/U a AH od 2008/09, **xG od 2023/24 (top 5) a 2025/26 (ostatní)**, uzavírací kurzy, ~30–120 sázkovek na zápas. Pinnacle a Betfair chybí. 1 000+ soutěží; Chance Liga nejmenována, nutno otestovat. | $5 / 1 000 zápasů |
| `zayne21/football-odds-comparison` | 6 | Flashscore odds feed, 160+ sázkovek ve 40+ zemích, **Česko volitelné jako země**, ale české sázkovky nejmenované. 10 trhů × plný čas / 1. / 2. poločas. | $5 / 1 000 zápasů |
| `extractify-labs/flashscore-matches-odds` | 13 | Kurzy podle sázkovky, závisí na geolokaci serveru. | |

**Hypotéza k otestování (hodina práce, pár centů):** Flashscore je produkt
české Livesport a jeho česká verze ukazuje kurzy Tipsportu, Fortuny a
Betana. Jestli Flashscore actor s `country=CZ` vrátí tyto sázkovky, je to
druhá cesta ke kurzům CZ sázkovek vedle Odds‑API.io. Není to potvrzené.

### Sestavy, lavička, střídání, zranění

| Actor | users/měs | Co dává |
|---|---|---|
| `blackfalcondata/sportradar-stats-scraper` | 3 | Veřejný widget sportradar.com: formace, základní jedenáctka s detaily, **lavička**, **střídání s minutou a kdo za koho**, časová osa, počet zranění. Příklady jen fotbal; hokej/Extraliga nepotvrzeny. $2 / 1 000 zápasů. |
| `incognito_mode/sofascore-match-analytics-scraper` | 10 | Statistiky, **xG, sestavy**. |
| `abotapi/sofascore-scraper` | 98 | Nejpoužívanější Sofascore scraper: skóre, statistiky, hráči, kurzy. |
| `harry.automazione/flashscore-injuries-suspensions` | 4 | **Zranění a tresty** z Flashscore. |
| `sports-odds-lab/transfermarkt-scraper` | 1 | Tržní hodnoty, smlouvy, zranění. |
| `constructive_calm/understat-football-analytics` | 11 | Understat xG a střely (jen top ligy). |
| `crawloop/fotmob-scraper`, `incognito_mode/fotmob-*` | 9 / 15 | FotMob: xG, ratingy, sestavy. |
| `ichigowa/sports-injury-reports`, `lergassy/sports-injuries-api` | 68 / 26 | Zranění NFL/NBA/MLB/**NHL**. |

Pro hokej: Sofascore i Flashscore Extraligu mají, takže sestavy a brankáře
by přes jejich scrapery šly; čas na ledě po lajnách ne.

### Predikční trhy a e‑sports

Kalshi a Polymarket: ~300 actorů, např. `lergassy/prediction-markets-api`
(27), `scrapemint/kalshi-prediction-market-scraper` (33). Kalshi má navíc
oficiální veřejné API, scraper není potřeba. E‑sports: jen `axlymxp/valorant-esports-scraper`
(VLR.gg) a `jungle_synthesizer/esports-tournament-roster-scraper`. Nic na
HLTV/CS2 kurzy.

## Apify MCP server

- Hostovaný na `mcp.apify.com`, přihlášení OAuth nebo `Authorization: Bearer <APIFY_TOKEN>`.
  Pro Claude Code existuje Apify plugin, který ho přibalí.
- Nástroje: `search-actors`, `fetch-actor-details`, `call-actor`,
  `get-dataset-items`. Umí tedy **dynamicky najít a spustit libovolný
  actor ze Store** bez předchozí konfigurace. Actory s plnými oprávněními
  jsou vyloučené.
- Limit 30 požadavků/s na uživatele. Účet Apify nutný (free plán má $5
  kreditu měsíčně), platí se pay‑per‑event actoru plus compute/proxy.
- V adresáři konektorů Claude.ai jsem Apify nenašel (hledání je částečné);
  dá se přidat jako vlastní MCP server.

## Právní rámec

- Všeobecné podmínky Apify: uživatel je **výhradně odpovědný** za legalitu
  dat (čl. 5.8), smí zpracovávat jen data, k nimž má oprávnění (6.2), a
  **odškodňuje Apify** za nároky třetích stran (11.1). Apify nikoho
  neodškodňuje; odpovědnost je stropovaná ročním plněním.
- Každý actor má disclaimer „nejsme spojeni s X, za dodržení podmínek X
  odpovídáte vy“. OddsPortal actor sám přiznává, že `robots.txt` cílového
  webu scraping zakazuje.
- Podmínky Tipsportu k automatizovanému stahování jsem nenašel online;
  ale to, že scraper neexistuje, nic nemění na pravidle v plánu.

**Závěr:** Apify neřeší právní otázku, outsourcuje techniku. Riziko zůstává
na nás a je stejné, jako kdybychom scraper napsali sami.

## Co to mění v plánu

1. **Fáze 1–3 (výzkum, backtest):** Apify je dobrá volba na **historická
   data**: OddsPortal (otevírací a uzavírací kurzy, 20 sezon, všechny
   trhy, hokej) a Flashscore (xG od 2023/24, uzavírací kurzy), plus
   Pinnacle closing lines. Řádově $10–30 za celý backtestový dataset místo
   stovek eur za API. Sestavy a střídání pro ablaci „absence/lavička“
   přes Sportradar widget a Sofascore. Toto je **jednorázový výzkumný
   sběr dat**, ne produkční závislost.
2. **Fáze 4+ (produkce):** cesta ke kurzům CZ sázkovek zůstává
   Odds‑API.io. Betano actor (39 uživatelů celkem) a případný Flashscore
   CZ feed jsou **záložní**, ne primární. Produkt nesmí stát na actoru,
   který udržuje jeden vývojář pro deset lidí.
3. **Pravidlo „scraping sázkovek je mimo“ se zpřesňuje:** přímý scraping
   webů sázkovek zůstává mimo. Čtení agregátorů (OddsPortal, Flashscore,
   Sofascore) přes Apify je povolené **jen pro výzkumná data do backtestu**
   a s vědomím, že porušujeme jejich robots.txt / podmínky, což je
   akceptovatelné pro interní výzkum a neakceptovatelné pro produkt.
4. **Signál o trhu:** 590 actorů na „sports odds“, drtivá většina pod 20
   uživatelů měsíčně. Poptávka po kurzových datech od malých vývojářů je
   roztříštěná a nikdo na tom nevydělává. To je mimochodem další argument
   do oponentury (námitka 10).
