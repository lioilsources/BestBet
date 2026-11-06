# Provideři a fit — kam appka sedí a kam ne

Ověřeno 2026-11-04 (zdroje v `README.md`). Kde je údaj z marketingu
providera nebo z recenzního webu, je to řečeno. „Fit“ hodnotí tři věci:
smí český uživatel u providera legálně sázet, dostaneme jeho kurzy přes API
bez scrapingu, a dává náš model (rozdělení skóre z vlny + absencí) na jeho
trzích hranu.

Právní rámec jednou větou: v ČR smí nabízet kurzové sázky jen držitel
základního povolení MF (zákon 186/2016); reklama (včetně affiliate odkazů,
metodika Celní správy MI‑2025‑023) na kohokoli jiného je zakázaná a MF
weby nelicencovaných operátorů blokuje. Od 1. 1. 2024 spadají pod zákon
výslovně i mobilní a webové aplikace hazardních her. Naše appka hazardní hrou
není (nepřijímá sázky), ale **zobrazení kurzu s odkazem** je reklama, a ta
má pravidla.

## A. České licencované sázkovky (cílová množina)

| Operátor | Postavení | Trhy, které nás zajímají | API / kurzy | Affiliate | Fit |
|---|---|---|---|---|---|
| **Tipsport** (+ **Chance**, stejná skupina) | Skupina si připisuje ~¾ českého trhu kurzových sázek; tržby 2025 ~16 mld. Kč, zisk ~3,6 mld. Chance 3,8 mld. tržby. | Nejširší nabídka na trhu: hendikepy, třetiny, poločasy, přesné výsledky, statistiky (rohy, karty). Hokejové třetinové trhy jsou tu nejhlubší v ČR. | Žádné veřejné API. **Odds‑API.io přidalo Tipsport.cz v 12/2025** — jediná legální cesta k jeho kurzům, kterou jsme našli. Ověřit pokrytí trhů (třetiny?) před fází 4. | Tipsport Partners: víceúrovňový rev‑share, bez CPA (StatsDrone, nepotvrzeno). Chance 30–50 % NGR. | **Jádro.** Bez Tipsportu appka pro českého sázkaře nedává smysl. Riziko: závislost na jednom agregátoru. |
| **Fortuna** | Dvojka; starší údaj 31 % podílu (dnes pravděpodobně méně). Penta. | Standardní rozsah, BetBuilder, Cash Out; užší než Tipsport. | Žádné API. V agregátorech nepotvrzeno. | Rev‑share 10–70 %, bez CPA (StatsDrone). | Chceme, ale **podmíněno** dostupností kurzů přes agregátor. Jinak v MVP jen jako „otevřít u Fortuny“ bez kurzu. |
| **Betano** (Kaizen Gaming, část vlastní Komárek) | Nejrychleji rostoucí; sponzor Chance Ligy, Sparty, Plzně, Komety. Podíl malý, ale růst agresivní. | Hodně trhů na vybrané zápasy, MatchCombo, BetBuilder, kombinace dřív vylučujících se trhů. Kurzy spíš nižší. | Žádné API. Globálně je Betano v některých agregátorech (jiné země); CZ ověřit. | NGR rev‑share, CPA na vyžádání. | Dobrý fit pro **přesné výsledky a kombinace** — tam je marže vysoká a emocionální sázkař tam chodí. Pro nás druhý nejdůležitější po Tipsportu. |
| **Sazkabet** (Allwyn/Sazka) | Vstup 2023, podíl malý. | Standardní. | Žádné API. | Neověřeno. | Doplněk. Jen pokud je v agregátoru. |
| **MerkurXtip**, **SynotTip** a další držitelé povolení | Malí hráči. | Standardní. | Ne. | — | Mimo MVP. Seznam licencí vždy brát z webu MF, ne z paměti. |

Co z toho plyne pro plán: appka bude v MVP umět „kurz u tvé sázkovky“ jen
tam, kde je kurz v Odds‑API.io (dnes potvrzen Tipsport). U ostatních ukáže
náš férový kurz a tlačítko „otevřít“. To je přijatelné, ale musí to být
řečeno uživateli předem.

## B. Globální sázkovky (reference, ne cíl)

| Operátor | Co je | Pro nás |
|---|---|---|
| **Pinnacle** | Nejostřejší kurzy, marže ~2 % na top ligách, neomezuje výherce. API **zavřené pro veřejnost od 23. 7. 2025**, jen na žádost s financovaným účtem a pro B2B/akademiky. V ČR nelicencován. | **Benchmark** modelu (fáze 1–3) a referenční uzavírací kurz pro CLV. Bereme přes OddsPapi / Odds‑API.io. Nikdy neukazujeme uživateli jako cíl sázky. |
| **Betfair Exchange** | Burza, cena = skutečná tržní pravděpodobnost bez marže (jen komise). Oficiální API s app key, ale pro CZ rezidenty nepřístupné (bez licence). | Druhá reference pro CLV. Stejný režim jako Pinnacle. |
| **bet365**, **Unibet/Kindred**, **Entain** (bwin, Ladbrokes), **Flutter** (Paddy Power, Sky Bet, FanDuel), **DraftKings**, **evoke** (William Hill, 888) | Objemoví hráči, bet365 nejširší nabídka na světě. V ČR nikdo z nich licenci nemá. | Jen jako další referenční kurzy na konsensus (medián trhu). Pro českého uživatele irelevantní a propagace zakázaná. |
| **1xBet**, **22bet**, offshore „CZ friendly“ | Na blacklistu MF. | Nikdy. Ani jako reference — jejich kurzy jsou šum. |

## C. E‑sports

| Hráč | Co je | Pro nás |
|---|---|---|
| **GG.bet**, **Thunderpick** (krypto), **Rivalry** (Pinnacle do ní investovala 2023), **Pinnacle esports**, **Betway esports** | Specialisti. CS2 ≈ 57 % globálního e‑sports handle (Sharpr/Abios za 2025); dál LoL, Dota 2, Valorant. Odhady trhu 2026 14–18,5 mld. USD (placené reporty, metodiky se liší). Žádný z nich není v ČR licencován. | Český e‑sports sázkař sází u **Tipsportu / Fortuny**, které e‑sports mají. Takže cílová sázkovka se nemění. |
| Data: **Oddin.gg** (kurzy + data, B2B), **GRID**, **PandaScore**, **Abios/Sharpr** | Oficiální datová práva na turnaje; drahé, B2B smlouvy. | Hlavní překážka. Bez oficiálních dat nejsou sestavy ani „stand‑in“ informace včas. |

**Fit modelu je paradoxně výborný:** v e‑sportu je „absence hráče“
(stand‑in, roster change po roster locku) a „forma na konkrétní mapě“
přesně to, co vlna + absence modelují, a trh je méně efektivní než fotbal.
**Fit byznysu je špatný:** data jsou drahá a licencovaná, Tipsport kurzy
na e‑sports přes agregátor nemáme potvrzené, a cílová skupina (18–25) je
nejcitlivější regulatorně. **Fáze 8, nejdřív CS2, až po 6 měsících provozu.**

## D. Krypto sázkovky a on‑chain protokoly

| Hráč | Co je | Pro nás |
|---|---|---|
| **Stake.com** (top 3 globálně dle Azuro blogu, Curaçao), **Cloudbet**, **Sportsbet.io**, **Rollbit**, **BC.Game** | Centralizované krypto sázkovky, bez CZ licence, bez (nebo s lehkým) KYC. MF je blokuje. | **Nula.** Propagace je trestná reklama na nelegální hazard; MF v 2026 výslovně varovalo influencery. Ani odkaz, ani kurz. |
| **Azuro** (Polygon, Gnosis, Base, Arbitrum; sdílený LP pool, frontendy třetích stran), **Overtime Markets**, **SX Bet** | On‑chain sázení, kurzy veřejně čitelné z chainu (Azuro má partnerství s Pinnacle Solution na pricing). | Technicky zajímavé: **veřejná, bezplatná, historizovaná kurzová data**. Právně stejné jako výše — uživatel by sázel u nelicencovaného operátora, MiCA po 1. 7. 2026 navíc dotahuje poskytovatele krypto služeb. Dá se použít jako **další referenční cena** v backtestu, nikdy jako cíl. |

Krypto „Originals“ (coinflip, crash, dice, plinko) jsou čisté RNG. Žádná
informace, žádný model, žádný fit. Uvádím to, protože „pravděpodobnostní
ruleta“ se často myslí právě tohle.

## E. Predikční trhy a „A/B rulety“

Interpretace zadání: „pravděpodobnostní A/B ruleta“ = binární trh
(stane se A, nebo B), kde cena = pravděpodobnost. To jsou predikční trhy.

| Hráč | Stav 2026 | Pro nás |
|---|---|---|
| **Kalshi** (CFTC‑regulovaná, USA) | Objem 2026 explodoval: s Polymarketem dohromady ~24 mld. USD/měs na jaře, **44,8 mld. USD v červnu 2026** (MS ve fotbale). Sport = 80–87 % objemu Kalshi. Spory se státy USA, NCAA/NFL/NBA proti. | V EU nedostupná (geoblok, ESMA 9/2026 kritizuje, že ne ve všech členských státech). |
| **Polymarket** (krypto, Polygon) | Druhý největší; sport ~40–46 % objemu. Francie (ANJ, 7/2026), Španělsko (5/2026), Portugalsko blokují; Německo zakázáno. ESMA 7/2026: event kontrakty mohou být finanční nástroje → **zákaz binárních opcí pro retail v EU**; tokenizované → MiCA. | ČR explicitně neblokuje (zatím), ale směr EU je jasný. Pro českého uživatele právně šedá až černá zóna. |
| **Manifold** (play‑money Mana) | Sweepcash zrušen 3/2025, zpět na čistě herní měnu. | Zajímavé jako **kalibrační hřiště**: zdarma, legální, sportovní trhy existují. Můžeme tam **publikovat predikce modelu** a budovat track record veřejně. Nulová monetizace. |
| **Fliff**, **Futuur**, sweepstakes/social | Fliff podává žádost o licenci FCM (8/2026) a pivotuje na event kontrakty. Sweepstakes model pod tlakem v USA. | US only. Sledovat jako produktový vzor „sociální A/B picks“, ne jako cíl. |
| **PrizePicks / Underdog** (pick’em more/less) | DFS pick’em, hráčské statistiky. US. | Model by potřeboval hráčské propy; mimo rozsah. |

**Fit modelu: nejlepší ze všech.** Binární trh bez marže je přesně to, co
pravděpodobnostní model potřebuje; hrana se měří přímo. **Fit práva: nejhorší
ze všech v EU 2026.** Rozhodnutí: nic z toho v produktu. Jediné dvě
použití: (1) ceny Polymarketu/Kalshi jako další reference v backtestu (jsou
veřejné), (2) Manifold jako veřejný track record. Přehodnotit, až bude
regulace EU hotová (ESMA avizuje další kroky).

## F. Analytické a tipovací nástroje (skutečná konkurence)

| Nástroj | Co dělá | Co dělá lépe než my | Co nedělá |
|---|---|---|---|
| **Sofascore**, **Flashscore** (Livesport, česká firma) | Live skóre, sestavy, ratingy hráčů, „win probability“ zdarma, obrovský dosah. | Data, rychlost, dosah, zdarma. Flashscore je domácí a má Tipsport integraci. | Nemají hranu vůči kurzu, CLV, deník. Pravděpodobnost je ozdoba, ne produkt. |
| **Forebet**, **Kickform**, **Betegy** | Statistické predikce fotbalu, od 2009, free + premium. | Pokrytí lig, SEO. | Netransparentní úspěšnost, žádné absence/sestavy v modelu, žádný CZ kurz. |
| **Trefík** (CZ) | Desktopový software pro sázkaře, Poisson, archiv kurzů od 1996, uživatel si ladí parametry. | Hloubka pro nerdy, historie, poctivost („žádné jisté tipy“). | Mobil, UX, sestavy a absence, automatizace. Je to tabulkový editor pro sázení. |
| **OddsJam**, **Rithmm**, **Outlier**, **Dimers**, **BetQL** (USA) | Value/arbitráž skener nad US kurzy, AI modely, předplatné 30–100 USD/měs. | Produktová zralost, kurzy přes oficiální feedy. | US kurzy, US sporty. Do ČR nikdy nepřijdou. |
| **Pikkit**, **Juice Reel**, **Bettingmetrics** | Deník sázek, sync s účty u sázkovek (USA), CLV. | Deník jako produkt, automatický import. | CZ sázkovky, model. |
| **Tipsport „Tipéři“**, komunity, Telegram tipaři | Sociální tipy, emoce, dosah. | Komunita, zdarma, uvnitř sázkovky. | Číslo. Poctivost. |

**Mezera, kterou nikdo nezaplňuje:** mobilní, český, kurzy českých sázkovek,
model se sestavami a absencemi, a deník s CLV. Každá část existuje někde,
dohromady nikde. (Zda je to mezera, nebo hrob, řeší `03_OPONENTURA.md`.)

## G. Souhrnná fit matice

| Skupina | Legální pro CZ uživatele | Kurzy přes API | Trhy HC / třetina / poločas | Hrana modelu | Verdikt |
|---|---|---|---|---|---|
| CZ licencované | ano | Tipsport ano (Odds‑API.io), ostatní ověřit | ano, Tipsport nejhlubší | malá, nutno prokázat (G1) | **MVP** |
| Globální sázkovky | ne | ano (agregátory) | ano | — | **reference** |
| E‑sports specialisti | ne (CZ e‑sports u Tipsportu) | částečně | ano (mapy, rounds) | potenciálně větší | **fáze 8** |
| Krypto CEX | ne, blokováno | ne | ano | — | **nikdy** |
| On‑chain (Azuro…) | ne | ano, veřejně | omezeně | — | **jen data do backtestu** |
| Predikční trhy | šedá/černá (ESMA) | ano, veřejně | binární | největší | **sledovat, nepoužít** |
| Play‑money (Manifold) | ano | ano | binární | — | **veřejný track record** |
| Analytické nástroje | — | — | — | — | **konkurence**, viz 04 |
