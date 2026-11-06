# Srovnání dvou týmů v den zápasu — specifikace

Zadání (2026-11-05): side‑by‑side srovnání dvou týmů, které proti sobě
nastoupí, s návrhy a zvýrazněním statistik, které dávají jednomu týmu
konkurenční výhodu.

Obrazovka je druhá záložka detailu zápasu (`01_PLAN_PRO_OPUS.md` §1,
obrazovka 1 „Zápas“ má záložky *Vlny*, **Srovnání**, *Trhy*). Čerpá
z datového modelu týmu (`06_DETAIL_TYMU.md`) a z výstupu modelu zápasu.
Nic nového nepočítá, jen vybírá a ukazuje.

## 1. Princip: dvě vrstvy zvýraznění

Každá statistika na obrazovce může být zvýrazněná dvěma různými způsoby a
uživatel musí na první pohled vidět, který to je:

| Vrstva | Barva / značka | Co znamená | Odkud |
|---|---|---|---|
| **Model** (plný štítek) | plná barva týmu, ikona vlny | Model zápasu tuhle věc **skutečně používá** a přispívá k odhadu ve prospěch týmu. Číslo u štítku je příspěvek v procentních bodech k pravděpodobnosti výhry. | Dekompozice příspěvků z modelu (§4) |
| **Fakt** (obrysový štítek) | obrys, bez ikony | Statisticky významný rozdíl mezi týmy, ale model ho nepoužívá (bilance na stadionu, historie H2H, trofeje…). Je to zajímavé, ne prediktivní. | Test rozdílu proti ligovému rozdělení (§3) |

Pravidlo: **žádná položka nemá plný štítek, pokud není v modelu.** To je
ochrana proti tomu, aby obrazovka vypadala jako seznam důvodů, proč vsadit.
Když model říká 52 : 48 a obrazovka svítí osmi „výhodami“ pro domácí, je
to lež; vrstvy to řeší tím, že fakt a model vypadají jinak.

## 2. Layout

Dvě sloupce, tým A vlevo, tým B vpravo, metriky uprostřed. Řádek metriky:
hodnota A | název metriky + ligový medián šedě | hodnota B. Vítěz řádku
má podbarvenou buňku (intenzita podle velikosti rozdílu, §3). Nahoře:

- **Hlavička**: loga, pozice v tabulce, forma 5 teček, **obě vlny vedle
  sebe** (stejný obrázek jako na záložce Vlny, menší), odhad modelu
  1 : X : 2 jako pruh.
- **„Co rozhoduje“**: 3–5 karet nahoře, výběr podle §4. Toto je ta část
  „návrhy a highlighty“. Každá karta: titulek (např. „Přesilovka Sparty
  proti oslabení Třince“), dvě čísla, počet zápasů/minut, štítek vrstvy,
  jedna věta.
- **Bloky metrik** (sbalitelné), pořadí podle sportu níž.
- Patička: „Zdroj dat, k datu, počet zápasů v sezoně“ a tlačítko
  „Trhy“.

### 2.1 Bloky metrik — hokej

| Blok | Řádky (A vs B) | Vrstva |
|---|---|---|
| Forma a síla | body/zápas posledních 10; vlna (úroveň, směr); xGF% 5v5 posledních 10; GF/60, GA/60 | model |
| Absence | kdo chybí, % síly týmu, dotčené dvojice (WOWY „partner chybí“, `06` §2.3) | model |
| Brankáři | startující (potvrzen/očekáván): SV%, GSAx/60, posledních 5 startů; dvojka a kdo chytal včera (back‑to‑back) | model |
| Speciální formace | PP% vs PK% soupeře, křížem; PP xGF/60 vs PK xGA/60; faulů za zápas (kolik přesilovek se dá čekat) | model |
| Pětky | první lajny LK–C–PK obou týmů s xGF% spolu; první obranné dvojice; buly centrů | fakt (do modelu jen přes `w` a `chem`) |
| Střelba | střel/60, kvalita střel (xG/střela), % střel z slotu kde je; blokované střely | fakt / částečně model přes xG |
| Prostředí | doma/venku bilance sezony; dny od posledního zápasu; venkovní tour (zápas č. N z M); cestování km | model (únava, domácí výhoda ligy) |
| Historie | H2H posledních 10; bilance A na stadionu B (s n) | fakt |
| Styl | tempo (střel obou týmů/60), podíl gólů ve 3. třetině, průměr gólů → relevantní pro třetinové trhy | fakt, částečně model (profil intenzity třetin) |

### 2.2 Bloky metrik — fotbal

| Blok | Řádky | Vrstva |
|---|---|---|
| Forma a síla | body/zápas posledních 10; vlna; xG − xGA/zápas posledních 10; dlouhodobá síla útok/obrana z Dixon–Coles | model |
| Absence | kdo chybí, % síly, náhradník a jeho `w`; dotčené dvojice (stoperská, střední záloha) | model |
| Sestava | očekávaná / potvrzená XI, formace proti formaci, počet změn vůči minulému zápasu | model (rotace) |
| Útok vs obrana křížem | xG/zápas A vs xGA/zápas B a naopak; střely v pokutovém území; standardky xG | model (přes útok/obrana) |
| Tempo a skóre | góly/zápas obou; podíl gólů v 1. poločase; BTTS % ; 0:0 % → relevantní pro poločasové, O/U a přesné výsledky | částečně model (poločasový profil) |
| Prostředí | doma/venku; dny odpočinku; evropský pohár ve středu; cestování | model |
| Historie | H2H posledních 10; bilance na stadionu (s n) | fakt |
| Disciplína | fauly, karty, penalty za/proti | fakt |

## 3. Kdy je rozdíl „výhoda“

Řádek dostane podbarvení jen když projde třemi filtry, všechny tři jsou
v kódu, ne v hlavě editora:

1. **Velikost účinku vůči lize**: rozdíl A − B vydělený směrodatnou
   odchylkou metriky napříč týmy ligy v sezoně (z‑rozdíl). Podbarvení
   začíná na |z| ≥ 0,5, plná intenzita od 1,5. Pod 0,5 je řádek neutrální,
   i když je A číselně „lepší“. Tím zmizí „11,2 vs 11,0 střel“ jako výhoda.
2. **Vzorek**: metrika má minimální n (zápasy nebo minuty), definované per
   metrika v konfiguraci (např. PP% ≥ 40 přesilovek, WOWY ≥ 100 min, H2H
   ≥ 6 zápasů, stadion ≥ 5 zápasů). Pod prahem šedý text „málo dat“ a
   nikdy podbarvení.
3. **Směr**: u metrik typu „méně je lépe“ (GA/60, karty) je vítěz obrácený.
   V konfiguraci metriky je `higher_is_better`.

Pro vrstvu **Fakt** stačí 1–3. Pro vrstvu **Model** musí navíc platit §4.

## 4. „Co rozhoduje“ — výběr karet

Zdroj pravdy je **dekompozice odhadu modelu**. Model zápasu (Dixon–Coles
+ vlna + absence + brankář + únava) umí pro každý vstup říct, o kolik
procentních bodů posouvá pravděpodobnost výhry A oproti stavu, kdy by
byl vstup na ligovém průměru (postupná substituce vstupů, Shapleyho
přibližná dekompozice přes permutace stačí; výpočet je levný, model je
malý).

Algoritmus výběru karet:

1. Vezmi příspěvky všech vstupů modelu, seřaď podle |příspěvek|.
2. Spoj vstupy, které patří k sobě, do jedné karty (PP A + PK B = jedna
   karta „speciální formace“; absence hráče + dotčené dvojice = jedna
   karta).
3. Vyber karty s |příspěvek| ≥ 1 p. b., maximálně 4. To jsou **plné**
   štítky. Pokud žádný vstup nedosáhne 1 p. b., první karta zní „Model
   nevidí výraznou výhodu; odhad X : Y : Z plyne z dlouhodobé síly“ — a je to
   legitimní, častý a poctivý výstup.
4. Doplň **nejvýš 1** kartu z vrstvy Fakt s nejvyšším |z| a dostatečným
   n, označenou obrysem. Nikdy víc než jednu, aby fakta nepřehlušila
   model.
5. Karty ukazují obě strany: pokud má A výhodu v přesilovce a B v
   brankáři, obě karty jsou vidět. Výběr není „důvody pro favorita“, je to
   „co se v tomhle zápase počítá“.

Každá karta má generovanou větu ze šablony per typ vstupu, vždy s čísly a
vzorkem, např.:

> **Brankář** · Třinec +2,3 p. b. · Kacetl 0,927 SV% / GSAx +0,41 za 60
> (24 startů) proti Hrubec 0,901 (19 startů). Model váží brankáře víc než
> kterýkoli jiný jeden vstup.

> **Absence** · Sparta −1,8 p. b. · Bez Chlapíka (14 % síly útoku) hraje
> 1. lajna 41 minut 5v5 s xGF% 44 % oproti 56 % s ním. Málo dat, odhad
> smrštěný.

Šablony jsou v konfiguraci, ne v kódu; jazyk cs/en.

## 5. Co obrazovka nedělá

- Nezobrazuje kurz. Kurzy a hrana jsou na záložce *Trhy*. Srovnání má být
  čitelné i pro člověka, který nesází.
- Neukazuje „doporučení“. Žádné „Vsaď A“, žádné „Náš tip“. Karty říkají,
  co se počítá, pruh nahoře říká, kolik to dohromady dělá.
- Nepoužívá H2H ani stadion v modelové vrstvě. Jsou to fakta.
- Nezvýrazňuje nic bez vzorku, nic pod |z| 0,5, a nikdy víc než 4 + 1
  karet. Když chce uživatel víc, rozbalí bloky.
- Nemá ranking „silnější tým“ v jiném čísle než v pravděpodobnosti
  modelu. Žádné skóre 7 : 4 „vyhrané řádky“, to je přesně ta iluze, kterou
  jiné appky prodávají.

## 6. API a výpočet

- `GET /matches/{id}/compare` vrací: hlavičku, obě vlny, pravděpodobnosti,
  pole `cards[]` (typ, strana, příspěvek p. b., vrstva, čísla, n, věta)
  a `blocks[]` s řádky (`metric_id`, `a`, `b`, `league_median`, `z`, `n_a`,
  `n_b`, `layer`, `winner`).
- Dekompozice příspěvků se počítá v `model/` při každém přepočtu zápasu a
  ukládá se vedle predikce (`MatchPrediction.contributions`). Appka nic
  nepočítá.
- Konfigurace metrik (`metrics.yaml`): id, název cs/en, sport, blok,
  zdrojové pole, `higher_is_better`, `min_n`, jednotka, formát. Přidání
  metriky = řádek v YAML + mapování v ingestu, bez změny UI.
- Ligové rozdělení (medián, směrodatná odchylka) se přepočítává denně per
  liga a sezona; prvních 5 kol sezony se mísí s minulou sezonou s vahou
  klesající podle počtu odehraných kol.

## 7. Zařazení do fází

| Co | Fáze |
|---|---|
| Dekompozice příspěvků v modelu (`contributions`) | **3** (patří k modelu trhů; zároveň je to vývojářský nástroj pro ablace) |
| `metrics.yaml`, ligová rozdělení, endpoint `/compare` | **4** |
| Obrazovka Srovnání s bloky Forma, Absence, Brankáři/Sestava, Prostředí a kartami „Co rozhoduje“ | **5** (MVP) — je to obrazovka, kterou uživatel otevře před každým zápasem |
| Pětky, WOWY, Styl, Historie, Disciplína | **6** |

## 8. Test akceptace

- Na 100 náhodných zápasech backtestu součet příspěvků karet ± zbytek
  („dlouhodobá síla“) dává pravděpodobnost modelu s odchylkou pod 0,5 p. b.
- Žádná karta s plným štítkem nemá vstup, který není v modelu (test nad
  konfigurací).
- Žádný podbarvený řádek s n pod prahem (test nad daty dne).
- U 20 zápasů ručně přečtené věty karet: čísla odpovídají blokům, vzorek
  je uvedený, nikde slovo „tip“, „sázka“, „jistota“.
