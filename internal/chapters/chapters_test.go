package chapters

import "testing"

func TestParse(t *testing.T) {
	ok := []struct {
		raw  string
		want float64
	}{
		{"311", 311},
		{"10.5", 10.5},
		{"1-5", 5},
		{"c.45", 45},
		{"v.3 c.12", 12},
		{"v.15 c.3", 3},
		{"Season 2 Ep. 4", 4},
		{"S2 - 15", 15},
		{"Vol.2 Ch.10-12", 12},
		{"c.100 (end)", 100},
	}
	for _, tc := range ok {
		got, found := Parse(tc.raw)
		if !found || got != tc.want {
			t.Errorf("Parse(%q) = %v, %v; want %v, true", tc.raw, got, found, tc.want)
		}
	}
	for _, raw := range []string{"", "Oneshot", "Prologue", "v.3"} {
		if got, found := Parse(raw); found {
			t.Errorf("Parse(%q) = %v, true; want not found", raw, got)
		}
	}
}

func TestNewCount(t *testing.T) {
	cases := []struct {
		latest, mine float64
		want         int
	}{
		{248, 212, 36},
		{212, 212, 0},
		{0, 50, 0},
		{10.5, 10, 0},
		{11, 10.5, 1},
		{100, 120, 0},
	}
	for _, tc := range cases {
		if got := NewCount(tc.latest, tc.mine); got != tc.want {
			t.Errorf("NewCount(%v, %v) = %d, want %d", tc.latest, tc.mine, got, tc.want)
		}
	}
}

func TestEffective(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	if got := Effective(311, nil); got != 311 {
		t.Errorf("Effective(311, nil) = %v", got)
	}
	if got := Effective(311, f(320)); got != 320 {
		t.Errorf("Effective(311, 320) = %v", got)
	}
	if got := Effective(330, f(320)); got != 330 {
		t.Errorf("Effective(330, 320) = %v", got)
	}
}
