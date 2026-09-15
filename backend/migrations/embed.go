package migrations

import "embed"

// FS contains versioned SQL migrations.
//
//go:embed *.sql
var FS embed.FS
