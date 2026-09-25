//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestServices_TwoOfThreeAreServed(t *testing.T) {
	c := newUser(t, "services")
	c.must(http.StatusOK, "GET", "/events", nil)
	c.must(http.StatusOK, "GET", "/playlists", nil)
	for _, p := range []string{"/devices", "/delegations"} {
		if st := c.do("GET", p, nil).status; st != http.StatusNotFound {
			t.Fatalf("%s answers %d: control delegation exists now, give it its own e2e tests", p, st)
		}
	}
}

func TestServer_BackendIsTheSourceOfTruth(t *testing.T) {
	c := newUser(t, "truth")
	c.must(http.StatusOK, "PATCH", "/users/me", map[string]any{
		"music_preferences": map[string]any{"genres": []string{"dub"}},
	})
	pl := c.createPlaylist(map[string]any{"name": "persisted " + runID, "visibility": "private"})
	c.must(http.StatusCreated, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[4]})
	ev := c.createEvent(map[string]any{"name": "persisted " + runID})

	other := &client{t: t, ip: nextIP(), email: c.email}
	other.login(password)
	if !contains(other.must(http.StatusOK, "GET", "/playlists/mine", nil).list(t), pl) {
		t.Fatal("the playlist must be found from another session")
	}
	if rows, _ := other.playlist(pl); len(rows) != 1 || int64(rows[0]["track_deezer_id"].(float64)) != tracks[4] {
		t.Fatalf("the playlist content must come from the server, got %v", rows)
	}
	other.must(http.StatusOK, "GET", "/events/"+ev, nil)
	me := other.must(http.StatusOK, "GET", "/users/me", nil)
	if !strings.Contains(string(me.body), "dub") {
		t.Fatalf("the profile must come from the server, got %s", me.body)
	}
}

func TestAPI_EveryRouteIsDocumented(t *testing.T) {
	res, err := http.Get(apiURL + "/swagger/doc.json")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("the API reference must be served at /swagger/doc.json: %v", err)
	}
	var doc struct {
		Paths map[string]map[string]struct {
			Parameters []any          `json:"parameters"`
			Responses  map[string]any `json:"responses"`
		} `json:"paths"`
	}
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		t.Fatalf("decode swagger: %v", err)
	}
	res.Body.Close()

	src, err := os.ReadFile("../internal/api/router.go")
	if err != nil {
		t.Fatalf("read router: %v", err)
	}
	groups := map[string]string{"e": "", "protected": "", "ag": "/auth", "gg": "/auth/google"}
	routeRe := regexp.MustCompile(`\b(e|protected|ag|gg)\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"`)
	paramRe := regexp.MustCompile(`:(\w+)`)
	n := 0
	for _, m := range routeRe.FindAllStringSubmatch(string(src), -1) {
		path := paramRe.ReplaceAllString(groups[m[1]]+m[3], "{$1}")
		if path == "/health" || path == "/" || path == "/privacy" || strings.HasPrefix(path, "/swagger") {
			continue
		}
		n++
		op, ok := doc.Paths[path][strings.ToLower(m[2])]
		if !ok {
			t.Errorf("%s %s is served but not documented", m[2], path)
			continue
		}
		if len(op.Responses) == 0 {
			t.Errorf("%s %s documents no output", m[2], path)
		}
		if strings.Contains(path, "{") && len(op.Parameters) == 0 {
			t.Errorf("%s %s documents no input", m[2], path)
		}
	}
	if n < 40 {
		t.Fatalf("parsed only %d routes out of router.go; the pattern is out of date", n)
	}
}

func TestAPI_SpeaksJSON(t *testing.T) {
	c := newUser(t, "json")
	for _, r := range []resp{
		c.do("GET", "/users/me", nil),
		c.do("GET", "/events/00000000-0000-0000-0000-000000000000", nil),
		c.do("POST", "/playlists", map[string]any{}),
		anon(t).do("GET", "/users/me", nil),
	} {
		if ct := r.header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("every answer must be JSON, got %q for %s", ct, r)
		}
		if !json.Valid(r.body) {
			t.Fatalf("body is not valid JSON: %s", r)
		}
	}
	if r := c.doHeaders("POST", "/playlists", nil, http.Header{"Content-Type": {"application/json"}}); r.status != http.StatusBadRequest {
		t.Fatalf("an empty body must be a 400, got %s", r)
	}
	if r := c.do("POST", "/playlists", "not an object"); r.status != http.StatusBadRequest {
		t.Fatalf("a body that is not an object must be a 400, got %s", r)
	}
}

func TestAPI_TrackSearchThroughTheBackend(t *testing.T) {
	c := newUser(t, "search")
	rows := c.must(http.StatusOK, "GET", "/tracks/search?q=daft+punk", nil).list(t)
	if len(rows) == 0 {
		t.Fatal("a search for a well-known artist must return tracks")
	}
	for _, k := range []string{"title", "artist_name"} {
		if rows[0][k] == nil {
			t.Fatalf("a search result must carry %s: %v", k, rows[0])
		}
	}
	if u, _ := c.must(http.StatusOK, "GET", "/tracks/3135553/preview", nil).obj(t)["preview_url"].(string); !strings.HasPrefix(u, "https://") {
		t.Fatalf("a preview must be playable, got %q", u)
	}
}
