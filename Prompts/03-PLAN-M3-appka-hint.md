# Plán 03: rychlý běh do appky (hint pro sázkaře)

Stav 2026-11-08. Navazuje na plán 02, kde brána A1 neprošla (`docs/DECISIONS.md` 2026-11-08). Vlastník rozhodl, že appka vznikne hned jako **ukázka dat, práce s nimi a evaluace**. Appka nesází a nepřijímá sázky. Je to hint s přidanou hodnotou pro sázejícího a feedback vlastníka určí, co přidat a co přehodnotit.

## 1. Rozhodnutí vlastníka (2026-11-08)

- M3 se otevírá, i když G1 a A1 neprošly. Appka je informační nástroj: nesází, nepřijímá peníze, žádný affiliate.
- Přidaná hodnota není předpověď lepší než trh. Je to **informace z trhu**: rozdíl měkkých knih proti Pinnaclu, nejlepší cena napříč knihami, pohyb kurzu a forma týmů.
- Rozpočet zůstává 0 Kč.

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
