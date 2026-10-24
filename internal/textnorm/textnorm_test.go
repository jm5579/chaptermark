package textnorm

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Omniscient Reader's Viewpoint":  "omniscient readers viewpoint",
		"Omniscient Reader’s Viewpoint":  "omniscient readers viewpoint",
		"Pokémon":                        "pokemon",
		"  Solo   Leveling: Ragnarok! ":  "solo leveling ragnarok",
		"전지적 독자 시점":                      "전지적 독자 시점",
		"ＯＲＶ":                            "orv",
		"100%_done":                      "100 done",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
