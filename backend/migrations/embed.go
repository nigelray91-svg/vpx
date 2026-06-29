// Package migrations embeds the SQL migration files so they ship inside the
// binary and run automatically on startup.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
