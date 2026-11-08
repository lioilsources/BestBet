# DATA_M0 — pokrytí dat (hokej)

Vygenerováno `ingest-hockey report` 2026-11-07 21:30 UTC. Soubor je generovaný, ručně ho neupravuj; komentáře patří do `docs/M0.md`.

## Souhrn per liga

| Liga | Role | Sezon | Zápasů | S kurzy | S brankáři | Marže 1X2 | Marže ML | Knih | Join | Skóre M1 | Neprošlo |
|---|---|---|---|---|---|---|---|---|---|---|---|
| NHL | control | 4 | 4453 | 21 % | 100 % | 9,4 % | 5,1 % | 3 | 95,8 % | kontrola | knih 3 < 5; join 95,8 % < 98 % |
| HockeyAllsvenskan | fringe | 4 | 1352 | 20 % | 96 % | 10,6 % | 8,0 % | 5 | 97,9 % | — | join 97,9 % < 98 % |

## Doporučení pro M1

Žádná okrajová liga zatím neprošla branami (≥ 3 sezony, ≥ 5 knih, join ≥ 98 %). Viz sloupec „Neprošlo“.

## Sezony

| Liga | Sezona | Zápasů | S kurzy | S brankáři | Mimo základní část | Po 60 min nerozhodnuto |
|---|---|---|---|---|---|---|
| NHL | 2023/24 | 1400 | 0 | 1400 | 88 | 288 |
| NHL | 2024/25 | 1398 | 0 | 1398 | 86 | 291 |
| NHL | 2025/26 | 1394 | 698 | 1394 | 82 | 348 |
| NHL | 2026/27 | 261 | 249 | 261 | 0 | 59 |
| HockeyAllsvenskan | 2023/24 | 409 | 0 | 408 | 45 | 90 |
| HockeyAllsvenskan | 2024/25 | 415 | 0 | 414 | 51 | 103 |
| HockeyAllsvenskan | 2025/26 | 410 | 159 | 364 | 46 | 88 |
| HockeyAllsvenskan | 2026/27 | 118 | 118 | 117 | 0 | 28 |

## Trhy v kurzech

| Liga | Trh | Zápasů | Knih |
|---|---|---|---|
| nhl | 1x2 | 946 | 3 |
| nhl | btts | 256 | 2 |
| nhl | correct_score | 250 | 1 |
| nhl | ml | 948 | 3 |
| nhl | p1_1x2 | 256 | 2 |
| nhl | p1_total | 840 | 3 |
| nhl | puck_line | 904 | 3 |
| nhl | total | 904 | 3 |
| swe-allsvenskan | 1x2 | 263 | 5 |
| swe-allsvenskan | btts | 118 | 4 |
| swe-allsvenskan | correct_score | 118 | 3 |
| swe-allsvenskan | ml | 277 | 5 |
| swe-allsvenskan | p1_1x2 | 118 | 4 |
| swe-allsvenskan | p1_total | 229 | 5 |
| swe-allsvenskan | puck_line | 263 | 4 |
| swe-allsvenskan | total | 263 | 4 |

## Join klíče per zdroj

| Liga | Zdroj | Fixtures | Spárováno | Podíl |
|---|---|---|---|---|
| nhl | nhl | 4550 | 4550 | 100,0 % |
| nhl | oddspapi | 1076 | 1031 | 95,8 % |
| swe-allsvenskan | oddspapi | 861 | 843 | 97,9 % |

## Nespárované fixtures (max 200)

- nhl / oddspapi `id1500023462075780`: no game in window: Columbus Blue Jackets - Los Angeles Kings
- nhl / oddspapi `id1500023470557988`: no game in window: Colorado Avalanche - Vegas Golden Knights
- nhl / oddspapi `id1500023470557990`: no game in window: Vegas Golden Knights - Colorado Avalanche
- nhl / oddspapi `id1500023470557992`: no game in window: Colorado Avalanche - Vegas Golden Knights
- nhl / oddspapi `id1500023470557994`: no game in window: Vegas Golden Knights - Colorado Avalanche
- nhl / oddspapi `id1500023470557996`: no game in window: Montreal Canadiens - Carolina Hurricanes
- nhl / oddspapi `id1500023470558008`: no game in window: Montreal Canadiens - Carolina Hurricanes
- nhl / oddspapi `id1500023470558010`: no game in window: Carolina Hurricanes - Montreal Canadiens
- nhl / oddspapi `id1500023470558024`: no game in window: Vegas Golden Knights - Anaheim Ducks
- nhl / oddspapi `id1500023470558032`: no game in window: Minnesota Wild - Colorado Avalanche
- nhl / oddspapi `id1500023470558040`: no game in window: Colorado Avalanche - Minnesota Wild
- nhl / oddspapi `id1500023470558042`: unknown team: C2/C3 - Colorado Avalanche
- nhl / oddspapi `id1500023470558058`: unknown team: A3 - A2
- nhl / oddspapi `id1500023470558072`: no game in window: Buffalo Sabres - Boston Bruins
- nhl / oddspapi `id1500023470558074`: unknown team: Wc1 - A1
- nhl / oddspapi `id1500023470558088`: no game in window: Carolina Hurricanes - Vegas Golden Knights
- nhl / oddspapi `id1500023470558090`: unknown team: Western Conference Champion - Eastern Conference Champion
- nhl / oddspapi `id1500023470558104`: no game in window: Dallas Stars - Minnesota Wild
- nhl / oddspapi `id1500023470558106`: unknown team: C3 - C2
- nhl / oddspapi `id1500023470558108`: no game in window: Colorado Avalanche - Los Angeles Kings
- nhl / oddspapi `id1500023470558112`: no game in window: Los Angeles Kings - Colorado Avalanche
- nhl / oddspapi `id1500023470558120`: no game in window: Colorado Avalanche - Los Angeles Kings
- nhl / oddspapi `id1500023470558122`: unknown team: Wc3 - Colorado Avalanche
- nhl / oddspapi `id1500023470558136`: no game in window: Pittsburgh Penguins - Philadelphia Flyers
- nhl / oddspapi `id1500023470558138`: unknown team: M3 - M2
- nhl / oddspapi `id1500023470558140`: no game in window: Carolina Hurricanes - Ottawa Senators
- nhl / oddspapi `id1500023470558144`: no game in window: Ottawa Senators - Carolina Hurricanes
- nhl / oddspapi `id1500023470558152`: no game in window: Carolina Hurricanes - Ottawa Senators
- nhl / oddspapi `id1500023470558154`: unknown team: Wc2 - M1
- nhl / oddspapi `id1500023470558156`: no game in window: Carolina Hurricanes - Philadelphia Flyers
- nhl / oddspapi `id1500023470558160`: no game in window: Philadelphia Flyers - Carolina Hurricanes
- nhl / oddspapi `id1500023470558168`: no game in window: Carolina Hurricanes - Philadelphia Flyers
- nhl / oddspapi `id1500023470558170`: no game in window: Philadelphia Flyers - Carolina Hurricanes
- nhl / oddspapi `id1500023470558186`: unknown team: A2/A3 - Buffalo Sabres
- nhl / oddspapi `id1500023470558196`: no game in window: Edmonton Oilers - Anaheim Ducks
- nhl / oddspapi `id1500023470558200`: unknown team: P3 - P2
- nhl / oddspapi `id1500023470558216`: no game in window: Vegas Golden Knights - Utah Mammoth
- nhl / oddspapi `id1500023470558218`: unknown team: Utah Mammoth - P1
- nhl / oddspapi `id1500023470874552`: unknown team: Colorado Avalanche - Wc3
- nhl / oddspapi `id1500023470874554`: unknown team: Colorado Avalanche - Wc3
- nhl / oddspapi `id1500023470874556`: unknown team: Wc3 - Colorado Avalanche
- nhl / oddspapi `id1500023470874558`: unknown team: Wc3 - Colorado Avalanche
- nhl / oddspapi `id1500023470874560`: unknown team: Colorado Avalanche - Wc3
- nhl / oddspapi `id1500023470874562`: unknown team: Wc3 - Colorado Avalanche
- nhl / oddspapi `id1500023470874564`: unknown team: Colorado Avalanche - Wc3
- swe-allsvenskan / oddspapi `id1500041650412459`: no game in window: Nybro Vikings IF - Tingsryds AIF
- swe-allsvenskan / oddspapi `id1500041661294167`: no game in window: Sodertalje SK - Ostersunds IK
- swe-allsvenskan / oddspapi `id1500041668931494`: no game in window: IK Oskarshamn - Almtuna IS
- swe-allsvenskan / oddspapi `id1500041668931498`: no game in window: Sodertalje SK - Mora IK
- swe-allsvenskan / oddspapi `id1500041668932256`: no game in window: IF Bjorkloven - IK Oskarshamn
- swe-allsvenskan / oddspapi `id1500041668932260`: no game in window: BIK Karlskoga - Nybro Vikings IF
- swe-allsvenskan / oddspapi `id1500041668932288`: no game in window: IK Oskarshamn - IF Bjorkloven
- swe-allsvenskan / oddspapi `id1500041668932354`: no game in window: Nybro Vikings IF - BIK Karlskoga
- swe-allsvenskan / oddspapi `id1500041668932382`: no game in window: IF Bjorkloven - IK Oskarshamn
- swe-allsvenskan / oddspapi `id1500041668932488`: no game in window: BIK Karlskoga - Nybro Vikings IF
- swe-allsvenskan / oddspapi `id1500041668932490`: no game in window: MODO Hockey - AIK IF
- swe-allsvenskan / oddspapi `id1500041668932746`: no game in window: Sodertalje SK - IF Bjorkloven
- swe-allsvenskan / oddspapi `id1500041668932774`: no game in window: IF Bjorkloven - Sodertalje SK
- swe-allsvenskan / oddspapi `id1500041668932824`: no game in window: IF Bjorkloven - BIK Karlskoga
- swe-allsvenskan / oddspapi `id1500041668932826`: no game in window: BIK Karlskoga - IF Bjorkloven
- swe-allsvenskan / oddspapi `id1500041668932828`: no game in window: IF Bjorkloven - BIK Karlskoga
- swe-allsvenskan / oddspapi `id1500041668932852`: no game in window: IF Troja-Ljungby - Vasteras IK
- swe-allsvenskan / oddspapi `id1500041668932854`: no game in window: Vasteras IK - IF Troja-Ljungby
