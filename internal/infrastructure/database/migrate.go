package database

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com.br/lucas-mezencio/pdsi1/migrations"
)

// Migrate runs embedded SQL migrations in order.
func Migrate(ctx context.Context, db *sql.DB) error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, entry.Name())
	}

	sort.Strings(files)

	for _, name := range files {
		contents, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(contents)); err != nil {
			return err
		}
	}

	return nil
}
