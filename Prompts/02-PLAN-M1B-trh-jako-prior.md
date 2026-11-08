# 02 — Plán M1b: trh jako prior, informační okno brankáře

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
vyjít a 00 §3 to předvídalo: „Prior = trh. Model začíná z implikované
pravděpodobnosti Pinnacle a upravuje ji o sestavu a vlnu.“ Tahle věta se
nikdy neimplementovala. Jediná hypotéza, kterou G1 nevyvrátila, je 00 §7
řádek 1 a 3: hodnota existuje jen **v okně nové informace**, tedy mezi tím,
kdy se dozvíme brankáře, a tím, kdy to trh zapracuje.

## 1. Rozhodnutí vlastníka (před startem)

| Otázka | Výchozí návrh |
|---|---|
| Směr | **A:** tento plán (trh jako prior + okno brankáře). Alternativy v §7. |
| Rozpočet | 0 Kč. Free tier OddsPapi a API-HOCKEY, veřejné NHL API. Placený tier až s výsledkem F1 a výslovným souhlasem. |
| Zdroj ohlášených brankářů NHL | Veřejné NHL API (`gamecenter/<id>/landing`, pokud nese startující brankáře) jako první volba; DailyFaceoff jen jako záloha a jen pro interní výzkum s cache (ToS šedá zóna, 00 §7 ř. 8). Vlastník potvrdí. |
| Časový rámec | F1 dva dny práce. F2 tři dny práce a **4–6 týdnů běhu** bez zásahu. F3 dva dny. |

## 2. Hypotézy (falzifikovatelné)

- **H1 Prior = trh.** Model `market-prior-v1`: logit odmaržované pravděpodobnosti
  trhu v čase t plus β·x vrstev (brankář, b2b), β fitované walk-forward.
  Předpověď: na kurzech t24h je log-loss aspoň o 0,5 % lepší než trh t24h sám.
- **H2 Horní mez hodnoty brankáře.** Když do t24h kurzu vložíme brankáře,
  který opravdu chytal (hindsight, tedy horní mez), přiblíží se log-loss
  uzavíracímu kurzu. Měří se, jaký díl mezery t24h → close brankář vysvětlí.
  Když je to méně než polovina a méně než 0,5 % absolutně, okno brankáře
  nemá cenu a směr A končí.
- **H3 Okno existuje.** Prospektivně: sázka v okamžiku ohlášení brankáře
  s modelem z H1 má CLV > 0 proti uzavíracímu Pinnaclu, 95 % interval nad nulou.

## 3. Fáze a brány

### F1 Retro horní mez (data už máme)

1. `internal/model/marketprior`: vstup odmaržovaná pravděpodobnost knihy
   (Shin, Pinnacle; bez Pinnacle medián knih), výstup 1X2 a ML. Vrstvy ze
   stávajícího `layers` beze změny; β fit na logitu, ridge 1, MinN 100.
2. `cmd/backtest -model market-prior -prior t24h|t1h|open`: walk-forward
   po týdnech jako dnes, hodnocení jen na odložených sezonách, jen NHL
   (Allsvenskan má kurzy u 217 zápasů, to na bránu nestačí).
3. Tři řádky výsledků per prior: trh sám, trh + b2b, trh + b2b + brankář
   (hindsight). Ke každému log-loss, Brier, bootstrap interval rozdílu.
4. `docs/RESULTS_M1B_F1.md`, zápis do DECISIONS.

**Brána A1:** H2 platí (brankář vysvětlí ≥ polovinu mezery t24h → close
a ≥ 0,5 % log-loss). Neplatí → směr A končí, plán se archivuje s výsledky.

### F2 Prospektivní sběr (JODA, cron)

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

Žádná appka ani UI. Žádné nové ligy (Maxa, DEL2, Extraliga) dřív než
projde A1. Žádné nové vrstvy baseline. Žádný placený tier bez souhlasu.
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

Doporučení: A. Je to nejlevnější test jediné hypotézy, která přežila, a po
F1 (dva dny) je jasné, zda stojí za 6 týdnů sběru.
