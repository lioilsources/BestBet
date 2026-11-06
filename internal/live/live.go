// Package live computes predictions for upcoming games (plan 01 fáze 4): fit
// the current model on every game played before now and predict the scheduled
// games inside the horizon. Pure over canon; cmd/predict stores the result.
package live

import (
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

// MinTrain mirrors the backtest: fewer played games and the league is skipped.
const MinTrain = 150

// Predict returns predictions for unplayed games starting in [now, now+horizon).
// Training uses only games that started before now.
func Predict(matches []canon.Match, now time.Time, horizon time.Duration, cfg periods.Config) ([]canon.Prediction, error) {
	var train, upcoming []canon.Match
	for _, m := range matches {
		switch {
		case m.StartAt.Before(now):
			train = append(train, m)
		case m.StartAt.Before(now.Add(horizon)) && !m.Played():
			upcoming = append(upcoming, m)
		}
	}
	played := 0
	for _, m := range train {
		if m.Played() {
			played++
		}
	}
	if len(upcoming) == 0 || played < MinTrain {
		return nil, nil
	}
	model, err := periods.Fit(train, now, cfg)
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
