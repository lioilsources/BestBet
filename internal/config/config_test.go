package config

import "testing"

func TestLoadRepoConfig(t *testing.T) {
	f, err := Load("../../config/leagues.yaml")
	if err != nil {
		t.Fatal(err)
	}
	l, ok := f.League("swe-allsvenskan")
	if !ok || len(l.Swehockey) == 0 || l.OT.Playoff != "5v5_20" {
		t.Fatalf("%+v", l)
	}
	if _, ok := f.League("nhl"); !ok {
		t.Fatal("nhl control league missing")
	}
}
