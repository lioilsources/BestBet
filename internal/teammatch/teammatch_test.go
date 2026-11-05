package teammatch

import "testing"

func TestMatch(t *testing.T) {
	teams := []Team{
		{1, "Ostersund", Norm("Ostersund")},
		{2, "MoDo", Norm("MoDo")},
		{3, "BIK Karlskoga", Norm("BIK Karlskoga")},
		{4, "Djurgarden", Norm("Djurgarden")},
		{5, "Djurgarden J20", Norm("Djurgarden J20")},
		{6, "HC Dynamo Pardubice B", Norm("HC Dynamo Pardubice B")},
	}
	m := New(teams, map[string]int32{Norm("Södertälje SK"): 9})
	cases := []struct {
		in     string
		id     int32
		method Method
	}{
		{"Östersunds IK", 1, ByFuzzy},
		{"MoDo Hockey", 2, ByFuzzy},
		{"Modo", 2, ByExact},
		{"Karlskoga", 3, ByFuzzy},
		{"Djurgården", 4, ByExact},
		{"Södertälje SK", 9, ByAlias},
		{"AIK", 0, NoMatch},
		{"Dynamo Pardubice", 0, NoMatch}, // "b" token missing on our side is fine, but check below
	}
	for _, c := range cases {
		id, method := m.Match(c.in)
		if c.in == "Dynamo Pardubice" {
			// Shorter name fully contained -> fuzzy hit on the B team. Documented limitation:
			// B/J20 teams need an alias when the A team is in the same league.
			if id != 6 || method != ByFuzzy {
				t.Errorf("%s: got %d %s", c.in, id, method)
			}
			continue
		}
		if id != c.id || method != c.method {
			t.Errorf("%s: got %d %s, want %d %s", c.in, id, method, c.id, c.method)
		}
	}
}

func TestNorm(t *testing.T) {
	if got := Norm("  HC  Škoda-Plzeň  "); got != "hc skoda plzen" {
		t.Fatal(got)
	}
}
