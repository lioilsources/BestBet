// Package features computes model inputs from canonical matches (plan 01 §2.1):
// the wave (EWMA level, trend, dispersion of performance), absences (w[p]),
// bench depth, fatigue and travel, play-off phase. Pure functions over canon
// structures; no database, HTTP or time.Now. Filled in phase 2 (M1).
package features
