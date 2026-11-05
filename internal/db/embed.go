// Package db holds the schema migrations (embedded) and the sqlc-generated queries in dbq.
package db

import "embed"

// Migrations are golang-migrate files; never edit a deployed one, add a new one.
//
//go:embed migrations/*.sql
var Migrations embed.FS
