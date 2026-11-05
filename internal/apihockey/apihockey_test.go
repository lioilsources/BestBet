package apihockey

import (
	"encoding/json"
	"testing"
)

const gamesJSON = `[
 {"id":1,"timestamp":1758301200,"week":"Regular Season","status":{"short":"FT"},
  "teams":{"home":{"id":10,"name":"MoDo"},"away":{"id":11,"name":"Ostersund"}},
  "scores":{"home":3,"away":1},
  "periods":{"first":"0-0","second":"1-1","third":"2-0","overtime":null,"penalties":null}},
 {"id":2,"timestamp":1758301200,"week":"Regular Season","status":{"short":"AP"},
  "teams":{"home":{"id":12,"name":"Mora"},"away":{"id":13,"name":"Bjorkloven"}},
  "scores":{"home":1,"away":2},
  "periods":{"first":"0-0","second":"0-1","third":"1-0","overtime":"0-0","penalties":"1-3"}},
 {"id":3,"timestamp":1758301200,"week":"Play Offs","status":{"short":"AOT"},
  "teams":{"home":{"id":12,"name":"Mora"},"away":{"id":13,"name":"Bjorkloven"}},
  "scores":{"home":2,"away":1},
  "periods":{"first":"1-0","second":"0-0","third":"0-1","overtime":"1-0","penalties":"0-0"}}
]`

func TestGameResult(t *testing.T) {
	var games []Game
	if err := json.Unmarshal([]byte(gamesJSON), &games); err != nil {
		t.Fatal(err)
	}
	want := []struct {
		decided string
		fh, fa  int
		rh, ra  int
	}{{"reg", 3, 1, 3, 1}, {"so", 1, 2, 1, 1}, {"ot", 2, 1, 1, 1}}
	for i, g := range games {
		r, err := g.Result()
		if err != nil {
			t.Fatalf("game %d: %v", g.ID, err)
		}
		w := want[i]
		if r.DecidedIn != w.decided || r.Final.Home != w.fh || r.Final.Away != w.fa || r.Reg.Home != w.rh || r.Reg.Away != w.ra {
			t.Errorf("game %d: %+v", g.ID, r)
		}
		if *g.Scores.Home != r.Final.Home || *g.Scores.Away != r.Final.Away {
			t.Errorf("game %d: derived final disagrees with scores", g.ID)
		}
	}
	if Status("AOT") != "finished" || Status("NS") != "scheduled" || Status("POST") != "postponed" {
		t.Error("status map")
	}
}

func TestAPIError(t *testing.T) {
	if apiError(json.RawMessage(`[]`)) != nil {
		t.Error("[] is no error")
	}
	if err := apiError(json.RawMessage(`{"plan":"Free plans do not have access to this season"}`)); err == nil {
		t.Error("object must be an error")
	}
}
