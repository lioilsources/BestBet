// Package explain decomposes a prediction into input contributions in
// percentage points of the home-win probability (07 §4): permutation Shapley
// over model inputs, each replaced by its league-average value (Shapley).
// Layers on top of the baseline report their marginal effect given the rest.
package explain
