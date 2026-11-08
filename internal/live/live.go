// Package live computes predictions for upcoming games (plan 01 fáze 4): fit
// the current model on every game played before now and predict the scheduled
// games inside the horizon. Pure over canon; cmd/predict stores the result.
package live

import (
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/layers"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

// MinTrain mirrors the backtest: fewer played games and the league is skipped.
const MinTrain = 150

// Step is the walk-forward window that fills the out-of-sample baseline
// intensities the layers are fitted on (same as the backtest).
const Step = 7 * 24 * time.Hour

// Predict returns predictions for unplayed games starting in [now, now+horizon).
// Training uses only games that started before now. With layers, the baseline
// is first walked forward over the history so every training game carries the
// intensities predicted before it was played, as in the backtest.
func Predict(matches []canon.Match, now time.Time, horizon time.Duration, cfg layers.Config) ([]canon.Prediction, error) {
	var train, upcoming []canon.Match
	for _, m := range matches {
		switch {
		case m.StartAt.Before(now):
			train = append(train, m)
		case m.StartAt.Before(now.Add(horizon)) && !m.Played():
			upcoming = append(upcoming, m)
		}
	}
	if len(upcoming) == 0 || played(train, now) < MinTrain {
		return nil, nil
	}
	in := layers.BuildInputs(matches)
	if len(cfg.Layers) > 0 {
		if err := fillOOS(train, now, cfg.Base, in); err != nil {
			return nil, err
		}
	}
	model, err := layers.Fit(train, now, cfg, in)
	if err != nil {
		return nil, err
	}
	out := make([]canon.Prediction, 0, len(upcoming))
	for _, m := range upcoming {
		p, err := model.Predict(m)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func played(ms []canon.Match, before time.Time) int {
	n := 0
	for _, m := range ms {
		if m.Played() && m.StartAt.Before(before) {
			n++
		}
	}
	return n
}

// fillOOS fits the baseline every Step on the games before the window and
// predicts the window, which records each game's out-of-sample intensities.
func fillOOS(train []canon.Match, now time.Time, base periods.Config, in layers.Inputs) error {
	if len(train) == 0 {
		return nil
	}
	start := train[0].StartAt
	for _, m := range train {
		if m.StartAt.Before(start) {
			start = m.StartAt
		}
	}
	for t := start.Add(Step); t.Before(now); t = t.Add(Step) {
		if played(train, t) < MinTrain {
			continue
		}
		end := t.Add(Step)
		var hist, window []canon.Match
		for _, m := range train {
			switch {
			case m.StartAt.Before(t):
				hist = append(hist, m)
			case m.StartAt.Before(end):
				window = append(window, m)
			}
		}
		if len(window) == 0 {
			continue
		}
		bm, err := layers.Fit(hist, t, layers.Config{Base: base}, in)
		if err != nil {
			return err
		}
		for _, m := range window {
			if _, err := bm.Predict(m); err != nil {
				return err
			}
		}
	}
	return nil
}
