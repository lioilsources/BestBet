# Mock poskytovatelů

Dokud nejsou klíče k API-HOCKEY a OddsPapi, celá cesta ingest → backtest → sázky → brána G1 běží proti `internal/provmock`.

## Co mock je

- Deterministický svět z `config/leagues.yaml`: 12 týmů na ligu (`Mock <liga> 01` až `12`), dvoukolový rozpis každé sezony, výsledky z Poissonova modelu s pevnou silou týmů.
- Stejné JSON tvary, jaké parsují skuteční klienti: API-HOCKEY `/leagues`, `/games`, `/odds` a OddsPapi `/tournaments`, `/fixtures`, `/markets`, `/historical-odds`.
- Tři knihy: `mock-sharp` (marže 2,5 %), `mock-cz1` (6 %), `mock-cz2` (8 %). Otevírací kurz má chybu, která se ke startu zmenšuje, takže pohyb otevření → uzavření a CLV mají co měřit. Jméno skutečné knihy (`pinnacle`) se vrací jako `mock-pinnacle`.

## Jak spustit

```sh
make db
make mock-run   # DB bestbet_mock: migrate, leagues, games, oddspapi (60 dní), odds, snapshot, report → data/DATA_MOCK.md
DATABASE_URL=postgres://bestbet:bestbet@127.0.0.1:5455/bestbet_mock?sslmode=disable \
  ./bin/backtest -from 2024 -holdout 2026 -gate -sharp mock-sharp -books mock-cz1,mock-cz2
```

## Pravidla

- `MOCK_PROVIDERS=1` funguje jen proti databázi, jejíž jméno končí `_mock`. Do skutečné DB se mock nedostane.
- Čísla z mocku nejsou výsledky. Ověřují jen, že potrubí počítá: mock trh je postavený tak, že model proti `mock-sharp` prohrává.
- Až budou klíče: `.env` (`APIHOCKEY_KEY`, `ODDSPAPI_KEY`), bez `MOCK_PROVIDERS`, stejné příkazy proti `bestbet`.
