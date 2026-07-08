package database

import (
	"context"
	"database/sql"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/tests/testcontainers"
)

func indexExists(t *testing.T, db *sql.DB, indexName string) bool {
	t.Helper()
	var name string
	err := db.QueryRowContext(context.Background(),
		`SELECT indexname FROM pg_indexes WHERE indexname = $1`, indexName,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("pg_indexes lookup failed for %s: %v", indexName, err)
	}
	return true
}

// TestMigrations_DropFirebaseIDUniqueIndexes asserts that migration 0007
// removed the defence-in-depth partial unique indexes that were causing
// production 500s when consecutive POST /users requests both stored
// firebase_id as an empty string. The indexes were harmless once empty
// strings were coerced to NULL via sql.NullString, but they remained a
// landmine for any future code path that forgot the wrapping.
//
// Because Firebase Auth guarantees UID uniqueness upstream, the unique
// indexes are redundant. If/when cardinality justifies reintroducing
// uniqueness, that should be its own PR with empirical justification.
func TestMigrations_DropFirebaseIDUniqueIndexes(t *testing.T) {
	ctx := context.Background()
	db, _, cleanup := testcontainers.StartPostgresContainer(ctx)
	if db == nil {
		t.Skip("docker not available")
	}
	defer cleanup()

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	for _, name := range []string{
		"idx_users_firebase_id_unique",
		"idx_doctors_firebase_id_unique",
	} {
		if indexExists(t, db, name) {
			t.Fatalf("expected partial unique index %q to be dropped by migration 0007, but it still exists", name)
		}
	}
}
