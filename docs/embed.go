// Package docs embeds the result reports and decisions for the app's
// evaluation page (plan 03 §2).
package docs

import "embed"

// FS holds the reports.
//
//go:embed RESULTS_*.md DECISIONS.md
var FS embed.FS
