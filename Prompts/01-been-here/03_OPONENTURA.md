# Oponentura — proč to nemůže fungovat

Napsáno tak, aby to bolelo. Ke každé námitce je „co by muselo platit“, aby
padla, a odkaz na místo v plánu, které to má změřit. Námitky jsou seřazené
od těch, které zabijí model, přes ty, které zabijí byznys, k těm, které
zabijí autora.

## Námitky proti modelu

### 1. Trh už to ví. Rychleji než ty.

Absence hráče, startující brankář, nový trenér — to všechno sázkovky
zapracují do kurzu během sekund od zveřejnění, protože jim data dodává
Sportradar/Genius přímo z ligy, a protože na to mají tradery i algoritmy.
Naše „oficiální sestava 60 minut před výkopem“ přijde v okamžiku, kdy kurz
už hodinu stojí na novém čísle. Očekávané sestavy za €199/měs jsou odhad,
který pozdní změny nechytá. Model, který reaguje na informaci až po trhu,
nemá hranu — má zpoždění.

Akademika to potvrzuje z druhé strany: Moskowitz (Journal of Finance 2021,
>100 000 kontraktů ve čtyřech ligách) najde momentum a value efekty
v pohybu kurzu od otevření do uzavření, ale konstatuje, že jsou příliš malé,
než aby přežily transakční náklady.

**Muselo by platit:** náš model najde hranu *před* zveřejněním sestavy
(tj. v odhadu, kdo chybí a jak to bolí) a ta hrana přežije do uzavíracího
kurzu. **Měří:** brána G1, řádek CLV proti uzavíracímu Pinnacle. Jestli CLV
není kladné, námitka platí a končíme.

### 2. Marže sežere všechno, co najdeš.

Česká 1X2 marže je 5–8 %, přesný výsledek 15–25 %, třetiny a poločasy
10–15 %. Abys na přesném výsledku vydělal, musí tvůj model být o 20 % lepší
než trh *na trhu, kde je trh nejslabší proto, že mu na něm nikdo nekonkuruje
objemem*. Ty trhy jsou „emocionální“ přesně proto, že tam sázkovka
nejvíc vydělává. Appka, která nasměruje emocionální sázkaře na emocionální
trhy, je marketingové oddělení Tipsportu zdarma.

**Muselo by platit:** simulovaný yield při CZ kurzech (ne Pinnacle) kladný
na dvou odložených sezonách. **Měří:** G1, třetí řádek. Plán tomu čelí tím,
že u přesných výsledků a třetin má vyšší práh hrany a CLV filtr; jestli je
na těch trzích CLV záporné, appka je tam **neukáže zelené nikdy** a je
z toho jen kalkulačka „kolik tě to stojí“.

### 3. „Vlna“ je recency bias s hezkým jménem.

Literatura o hot hand a momentum v sázení (Camerer 1989, Brown & Sauer 1993,
Offerman & Sonnemans 2004) říká, že **sázkaři už teď přeceňují formu** a že
kurz formu částečně odráží. Model postavený na posledních deseti zápasech
může snadno jen zopakovat tu samou chybu — a pak bude souhlasit s davem a
nesouhlasit s Pinnaclem, což je přesná definice špatné sázky. Jediná
známá výjimka (Byrnes & Farinella, NBA, 56,5 % proti spreadu) je jedna
studie bez out‑of‑sample potvrzení.

**Muselo by platit:** vlna zlepšuje log‑loss proti holému Dixon–Colesovi
**a** reziduum modelu nekoreluje s pohybem kurzu od otevření k uzavření
(tj. vlna přidává informaci, kterou trh nemá, ne tu, kterou už zapracoval).
**Měří:** fáze 2, ablace a korelační test. Je explicitně v plánu, protože
to je nejpravděpodobnější způsob, jak se celá myšlenka rozpadne.

### 4. Nemáš data na „lavičku“ a „střídání“.

Dopad absence = síla hráče − síla náhradníka. Sílu hráče odhadneš z on/off
xG, což pro jednoho hráče v jedné sezoně znamená pár set minut a obří šum;
každý, kdo to zkoušel (RAPM, plus‑minus v hokeji), to ví. Pro Chance Ligu
nemáš ani pořádné xG, pro Extraligu nemáš polohu střel. Hloubka lavičky
v hokeji je čas na ledě po lajnách — to dává Sportradar za ceny, které
nezaplatíš. Výsledek: tři krásné parametry, z nichž dva budou regularizované
k nule a třetí bude šum.

**Muselo by platit:** ablace ve fázi 2 ukáže měřitelný přínos složky
„absence“ a „lavička“ alespoň v top 5 ligách, kde data jsou. Pro Chance Ligu
a Extraligu plán počítá s tím, že složka může jít ven, a MVP je pak jen
„vlna + Dixon–Coles“ — což je to, co má Forebet od 2009 zdarma.

### 5. Nikdy nebudeš vědět, jestli to funguje.

Hrana 3 % při kurzu 2,0 znamená, že potřebuješ řádově **tisíce** sázek, aby
95 % interval yieldu nezahrnoval nulu. Chance Liga má 240 zápasů za sezonu.
Extraliga 364. I s pěti ligami a dvěma sezonami je G1 na hraně statistické
síly; a to je backtest, ne živý provoz. Živě budeš po roce mít pár stovek
sázek a nebudeš vědět nic. Uživatel po 50 sázkách neví vůbec nic a odejde
podle toho, jak padly poslední tři.

**Muselo by platit:** přijmeš CLV jako primární metriku (konverguje
řádově rychleji než yield), a uživatel to přijme taky. Plán na tom stojí
(deník = CLV). Jestli uživatelé CLV nepochopí, námitka 11 platí dvojnásob.

### 6. Když to bude fungovat, přestane to fungovat.

Výherci dostanou stake factor. V UK je omezeno přes 4 % sázkařů (UKGC 2025),
Racing Post uvádí dva z pěti dotázaných. Spouštěč není výhra, ale **vzorec
chování**: sázky těsně po změně kurzu, na nelikvidní trhy, přesně tam, kde
appka pošle stovky lidí ve stejnou minutu. Tipsport uvidí tisíc účtů, které
sázejí stejný přesný výsledek 40 minut před výkopem, a omezí je všechny.
Appka vlastním úspěchem zničí účty svých uživatelů. A druhý řád: stejné
sázky od tisíce lidí pohnou kurzem a hrana zmizí, než ji vsadí ten tisící.

**Muselo by platit:** produkt zůstane malý (stovky, ne desítky tisíc
aktivních), nebo se přestane soustředit na „sázej teď“ a bude deníkem a
kalkulačkou. Plán to řeší stropem 5 doporučení/den a tím, že doporučení
nejsou push — ale neřeší to, co se stane, když se povede. To je férová
nevyřešená námitka.

## Námitky proti byznysu

### 7. Právo: nesmíš na to dělat reklamu a brzo možná nikdo.

Reklama na hazard v ČR: zakázána pro nelicencované (to nás netrápí),
a pro licencované s povinným varováním a bez cílení na mladé. V červnu 2025
byl na stole návrh zákazu tiskové reklamy a TV/radio reklamy 06–22 (online
mimo); osud neznáme, ale směr je jasný, sportovní kluby křičí o financování.
Appka, která ukazuje kurzy a má affiliate odkazy, **je** reklama na hazard a
Celní správa má od 7/2025 metodiku, jak affiliate kontrolovat. Google i Meta
reklamu na „betting tips“ v ČR prakticky neschválí. Apple a Google
vyžadují pro cokoli s kurzy 18+ a zemi‑specifickou dostupnost. Slovo
„emocionální“ v názvu nebo marketingu je dar regulátorovi.

**Muselo by platit:** růst čistě organický (YouTube rozbory, fóra), affiliate
jen k licencovaným a s varováním, a produkt se právně prezentuje jako
analytický nástroj. Plán s tím počítá, ale organický růst v nišovém
českém trhu je pomalý a to naráží na námitku 10 (break‑even).

### 8. Predikční trhy, krypto a e‑sports jsou v EU zavřené dveře.

Jediné prostředí, kde pravděpodobnostní model má čistou hranu (binární
trh bez marže), je právě to, které ESMA v 7/2026 posouvá pod zákaz
binárních opcí pro retail, Francie, Španělsko a Portugalsko blokují
Polymarket a Německo ho zakazuje. Krypto sázkovky jsou v ČR blokované a
jejich propagace trestná. E‑sports data jsou za B2B smlouvami. Všechny tři
„rozšiřující“ směry z zadání jsou pro českého uživatele buď nelegální,
nebo nedostupné. Zbývá Tipsport.

**Muselo by platit:** změna regulace EU směrem k povolení event kontraktů
(opak současného trendu), nebo relokace produktu mimo ČR (jiný produkt).

### 9. Závislost na jednom agregátoru a jedněch podmínkách.

Kurzy Tipsportu máme jen přes Odds‑API.io (přidáno 12/2025). Jeden
dodavatel, jeden ToS, jedna změna ceníku nebo jedno právní upozornění od
Tipsportu agregátorovi — a appka neumí svou hlavní větu „u tvé sázkovky je
kurz X“. Fortuna, Betano, Sazkabet nejsou potvrzené vůbec. Pinnacle zavřel
API v 7/2025; agregátory jsou další článek řetězu, který se může zavřít.
Scraping je zakázaný v plánu i v ToS sázkovek.

**Muselo by platit:** pokrytí ověřené před fází 4 a produkt navržený tak,
že bez kurzu operátora dál funguje (férový kurz + „otevřít“). Plán to má,
ale je to degradovaný produkt.

### 10. Čísla byznysu nevychází.

Data 300–500 €/měs, hosting, Apple/Google 15–30 %, právník. Break‑even
~150 předplatitelů po 9 €/měs v trhu, kde Tipsport drží ¾, Flashscore
(česká firma, miliony uživatelů) dává pravděpodobnosti zdarma a Trefík
obsluhuje nerdy 30 let. Cílová skupina „český sázkař, který chce čísla a ne
tipy a zaplatí za to“ je pár tisíc lidí; konverze 30 % z aktivních je
fantazie, realita u utilit je 2–5 %. Sázkaři platí za **tipy** (Telegram
tipaři berou tisíce měsíčně), ne za **pravděpodobnosti**. Produkt, který
odmítá říkat „vsaď si“, odmítá to jediné, za co segment platí.

**Muselo by platit:** retence D30 > 40 % u testovací skupiny (fáze 6) a
ochota platit potvrzená před fází 7. Plán to má jako bránu, ale přiznejme
si, že 10 přátel v TestFlightu není trh.

### 11. Kdo za to ponese odpovědnost, když uživatel prohraje.

Uživatel vsadí podle „zelené“ hrany, prohraje pětkrát za sebou (při hraně
3 % je to běžné), a napíše do recenze / na Reddit / Ministerstvu financí,
že appka ho navádí k sázení. Jestli model nefunguje, appka škodí. Jestli
funguje, uživatel to nepozná (námitka 5) a odejde stejně. Asymetrie:
negativní zkušenost je hlasitá a regulatorně nebezpečná, pozitivní je
statisticky neviditelná. Slovo „emoce“ v konceptu to zesiluje — cíleně
pracovat s emocemi sázkaře je přesně to, co regulátor a zákon o
odpovědném hraní zakazují.

**Muselo by platit:** produkt je „deník a kalkulačka“ s doporučeními jako
vedlejší funkcí, texty a UI prošly právníkem, a slovo „emoce“ zůstane
v interním názvu konceptu, ne v produktu.

## Námitky proti autorovi

### 12. Jeden člověk, dva produkty, žádný kvant.

Been Here je rozestavěný (fáze 4 a 5 čekají na projetí autem a kameru).
Tenhle plán je 21 týdnů práce člověka, který dělá model, backend, appku,
právo a marketing. Model vyžaduje statistiku na úrovni, kterou se člověk
učí roky (regularizace on/off, kalibrace, walk‑forward bez leaků, Shin
devig); chyby v backtestu (lookahead, survivorship, špatné uzavírací kurzy)
dávají falešně kladné výsledky a jsou tím pravděpodobnější, čím víc si
autor přeje, aby to vyšlo. Opus ty chyby udělá taky, jen rychleji.

**Muselo by platit:** fáze 1–3 (10 týdnů) jsou brané jako **výzkumný
projekt s předem napsaným kritériem neúspěchu**, ne jako první krok
k appce. Jestli G1 neprojde, byl to levný způsob, jak se to naučit. Jestli
projde, teprve pak se to stává produktem a tím se vrací námitky 6–11.

## Shrnutí oponenta

Model: pravděpodobně reprodukuje to, co trh ví, a na trzích, kde trh neví,
je marže větší než jakákoli realistická hrana. Byznys: právně svázaný,
závislý na jednom datovém dodavateli, v segmentu, který platí za tipy a ne
za čísla, proti domácímu gigantu, který má ¾ trhu. Autor: sám, bez kvanta,
s jiným rozestavěným produktem.

Co by oponent uznal: (a) deník s CLV je poctivý produkt a v ČR neexistuje;
(b) go/no‑go brána po 10 týdnech bez UI je správný způsob, jak tohle
zjistit levně; (c) e‑sports a predikční trhy jsou místa, kde model má
reálnou hranu, a stojí za to je sledovat pro den, kdy se regulace vyjasní.
