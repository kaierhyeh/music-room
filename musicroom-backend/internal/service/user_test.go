package service

import (
	"encoding/json"
	"testing"
)

func TestCanSee(t *testing.T) {
	cases := []struct {
		level string
		rel   Relationship
		want  bool
	}{
		{"public", RelOther, true},
		{"public", RelFriend, true},
		{"public", RelSelf, true},
		{"friends", RelOther, false},
		{"friends", RelFriend, true},
		{"friends", RelSelf, true},
		{"private", RelOther, false},
		{"private", RelFriend, false},
		{"private", RelSelf, true},
		{"", RelOther, false},
		{"", RelFriend, false},
		{"", RelSelf, true},
		{"bogus", RelFriend, false},
	}
	for _, c := range cases {
		if got := canSee(c.level, c.rel); got != c.want {
			t.Errorf("canSee(%q, %v) = %v, want %v", c.level, c.rel, got, c.want)
		}
	}
}

func TestFilterProfileData(t *testing.T) {
	data := map[string]json.RawMessage{
		"bio":      json.RawMessage(`"hello"`),
		"city":     json.RawMessage(`"Paris"`),
		"birthday": json.RawMessage(`"1995-03-12"`),
		"secret":   json.RawMessage(`"x"`),
	}
	vis := map[string]string{
		"bio":      "public",
		"city":     "friends",
		"birthday": "private",
	}

	if got := filterProfileData(data, vis, RelSelf); len(got) != 4 {
		t.Errorf("self should see all 4 fields, got %d", len(got))
	}
	assertKeys(t, "friend", filterProfileData(data, vis, RelFriend), "bio", "city")
	assertKeys(t, "other", filterProfileData(data, vis, RelOther), "bio")
}

func assertKeys(t *testing.T, name string, m map[string]json.RawMessage, want ...string) {
	t.Helper()
	if len(m) != len(want) {
		t.Errorf("%s: %d fields %v, want %v", name, len(m), keysOf(m), want)
		return
	}
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Errorf("%s: missing field %q", name, k)
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
