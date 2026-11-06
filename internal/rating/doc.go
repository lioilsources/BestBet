// Package rating holds slow-moving strength estimates (plan 00 §1.4.2):
// a Kalman team rating with velocity (the long wave) and goalie ratings
// (save percentage shrunk to the league mean by number of starts). Pure
// functions over canon structures. Filled in phase 2 (M1).
package rating
