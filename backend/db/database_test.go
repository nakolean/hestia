package db

import (
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	tables := []string{"chores", "shopping_items", "api_keys"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
		if err != nil {
			t.Fatalf("checking table %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("table %s was not created", table)
		}
	}
}
