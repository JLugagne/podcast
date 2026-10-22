// Package pgadapter implements the catalog persistence ports and the unit of
// work on PostgreSQL (pgx).
package pgadapter

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrations returns the goose migrations of the catalog context.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}
