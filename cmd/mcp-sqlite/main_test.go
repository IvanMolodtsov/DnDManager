package main

import "testing"

func TestReadOnlySQL(t *testing.T) {
	ok := []string{
		"SELECT * FROM characters",
		"with x as (select 1) select * from x",
		"PRAGMA table_info(characters)",
		"-- comment\nSELECT 1",
	}
	for _, q := range ok {
		if !readOnlySQL(q) {
			t.Fatalf("want allow %q", q)
		}
	}
	bad := []string{
		"INSERT INTO characters(name) VALUES ('x')",
		"UPDATE characters SET name='x'",
		"DELETE FROM characters",
		"DROP TABLE characters",
		"SELECT 1; DELETE FROM characters",
		"PRAGMA journal_mode",
	}
	for _, q := range bad {
		if readOnlySQL(q) {
			t.Fatalf("want deny %q", q)
		}
	}
}
