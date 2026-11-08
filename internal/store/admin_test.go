package store

import "testing"

func TestSplitDSN(t *testing.T) {
	name, admin, err := splitDSN("postgres://u:p@db.example:5455/bestbet_mock?sslmode=disable")
	if err != nil || name != "bestbet_mock" || admin != "postgres://u:p@db.example:5455/postgres?sslmode=disable" {
		t.Fatalf("got %q %q %v", name, admin, err)
	}
	if _, _, err := splitDSN("postgres://u:p@h/bad;name"); err == nil {
		t.Fatal("accepted a non-identifier database name")
	}
}
