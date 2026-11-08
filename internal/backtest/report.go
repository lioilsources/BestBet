package backtest

import (
	"fmt"
	"strings"
	"time"
)

// ReportInput is what RenderResults needs.
type ReportInput struct {
	Title        string
	League       string
	ModelVersion string
	Config       string // hyperparameters, one line
	Generated    time.Time
	Holdout      int // first held-out season (0 = none)
	Rows         []Row
}

// RenderResults renders a Czech RESULTS_*.md: one table per season plus total,
// market column always present ("chybí" when there are no odds).
func RenderResults(in ReportInput) (string, error) {
	total, err := Summarise(in.Rows)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# %s\n\n", in.Title)
	w("Liga `%s`, model `%s`, vygenerováno %s příkazem `backtest`. Hyperparametry: %s.\n\n", in.League, in.ModelVersion, in.Generated.UTC().Format("2006-01-02 15:04 UTC"), in.Config)
	w("**Závěr:** %s\n\n", conclusion(total))
	w("Log-loss 1X2 je za 60 minut (3 výsledky); nižší je lepší. „Základ“ = četnosti domácí/remíza/hosté z tréninkových dat. „Trh“ = odmaržovaný (Shin) medián knih před začátkem, na zápasech, kde kurzy jsou.\n\n")
	w("| Sezona | Zápasů | Log-loss model | Log-loss základ | Log-loss trh | Model na stejných | Brier | Log-loss ML | Remízy skutečně / model |\n")
	w("|---|---|---|---|---|---|---|---|---|\n")
	by := BySeason(in.Rows)
	for _, s := range Seasons(by) {
		m, _ := Summarise(by[s])
		label := fmt.Sprintf("%d/%02d", s, (s+1)%100)
		if in.Holdout != 0 && s >= in.Holdout {
			label += " (odložená)"
		}
		w("| %s | %s |\n", label, row(m))
	}
	w("| **celkem** | %s |\n", row(total))
	return b.String(), nil
}

func row(m Metrics) string {
	market, same := "chybí", "—"
	if m.NMarket1X2 > 0 {
		market = fmt.Sprintf("%s (n=%d)", num(m.MarketLogLoss1X2), m.NMarket1X2)
		same = num(m.ModelLLOnMarket)
	}
	return fmt.Sprintf("%d | %s | %s | %s | %s | %s | %s | %s / %s",
		m.N, num(m.LogLoss1X2), num(m.BaseLogLoss1X2), market, same, num(m.Brier1X2), num(m.LogLossML),
		pct(m.DrawRate), pct(m.MeanPDraw))
}

func conclusion(m Metrics) string {
	vsBase := 100 * (m.BaseLogLoss1X2 - m.LogLoss1X2) / m.BaseLogLoss1X2
	s := fmt.Sprintf("model je o %s %% lepší než základní četnosti (log-loss 1X2, %d zápasů)", num1(vsBase), m.N)
	if m.NMarket1X2 == 0 {
		return s + "; **srovnání s trhem chybí, protože pro tyto zápasy nejsou kurzy**, takže o hraně výsledek nic neříká."
	}
	gap := 100 * (m.ModelLLOnMarket - m.MarketLogLoss1X2) / m.MarketLogLoss1X2
	if gap > 0 {
		return s + fmt.Sprintf("; proti trhu je o %s %% horší (%d zápasů s kurzy).", num1(gap), m.NMarket1X2)
	}
	return s + fmt.Sprintf("; proti trhu je o %s %% lepší (%d zápasů s kurzy).", num1(-gap), m.NMarket1X2)
}

func num(f float64) string  { return strings.Replace(fmt.Sprintf("%.4f", f), ".", ",", 1) }
func num1(f float64) string { return strings.Replace(fmt.Sprintf("%.1f", f), ".", ",", 1) }
func pct(f float64) string  { return strings.Replace(fmt.Sprintf("%.1f %%", 100*f), ".", ",", 1) }
