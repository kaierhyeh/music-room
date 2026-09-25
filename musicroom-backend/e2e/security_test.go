//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSecurity_NoAccessToAnotherUsersData(t *testing.T) {
	alice := newUser(t, "secalice")
	mallory := newUser(t, "secmallory")

	for _, p := range []string{"/users/me", "/friends", "/events", "/playlists", "/playlists/mine", "/users/me/subscription"} {
		anon(t).must(http.StatusUnauthorized, "GET", p, nil)
	}

	alice.must(http.StatusOK, "PATCH", "/users/me", map[string]any{
		"profile_data":       map[string]any{"phone": "+33 6 00 00 00 00"},
		"profile_visibility": map[string]string{"phone": "private"},
	})
	seen := mallory.must(http.StatusOK, "GET", "/users/"+alice.id, nil).obj(t)
	if strings.Contains(string(mallory.must(http.StatusOK, "GET", "/users/"+alice.id, nil).body), "+33") || seen["email"] != nil {
		t.Fatalf("private data leaked: %v", seen)
	}

	if me := mallory.must(http.StatusOK, "GET", "/users/me?id="+alice.id, nil).obj(t); me["id"] != mallory.id {
		t.Fatalf("/users/me must answer for the token's owner only, got %v", me["id"])
	}

	pl := alice.createPlaylist(map[string]any{"name": "mine " + runID, "visibility": "private"})
	if contains(mallory.must(http.StatusOK, "GET", "/playlists/mine", nil).list(t), pl) {
		t.Fatal("someone else's playlist must not appear under /playlists/mine")
	}
	mallory.must(http.StatusForbidden, "GET", "/playlists/"+pl, nil)
	mallory.must(http.StatusForbidden, "PATCH", "/playlists/"+pl, map[string]any{"visibility": "public"})
	mallory.must(http.StatusForbidden, "GET", "/playlists/"+pl+"/collaborators", nil)
	mallory.must(http.StatusForbidden, "POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": mallory.id, "can_edit": false})

	ev := alice.createEvent(map[string]any{"name": "mine " + runID, "visibility": "private"})
	mallory.must(http.StatusForbidden, "GET", "/events/"+ev+"/invitations", nil)
	mallory.must(http.StatusForbidden, "POST", "/events/"+ev+"/invitations", map[string]any{"user_id": mallory.id})
}

func TestSecurity_TokensCannotBeForgedReplayedOrKeptAfterLogout(t *testing.T) {
	c := newUser(t, "secsession")
	parts := strings.Split(c.token, ".")
	if len(parts) != 3 {
		t.Fatalf("access token is not a JWT: %q", c.token)
	}

	forge := func(tok string) int {
		f := &client{t: t, ip: nextIP(), token: tok}
		return f.do("GET", "/users/me", nil).status
	}
	other := newUser(t, "secother")
	if st := forge(parts[0] + "." + strings.Split(other.token, ".")[1] + "." + parts[2]); st != http.StatusUnauthorized {
		t.Fatalf("a spliced token must be refused, got %d", st)
	}
	none := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	if st := forge(none + "." + parts[1] + "."); st != http.StatusUnauthorized {
		t.Fatalf("an alg=none token must be refused, got %d", st)
	}
	if st := forge("garbage"); st != http.StatusUnauthorized {
		t.Fatalf("a garbage token must be refused, got %d", st)
	}

	stolen := c.refresh
	r := c.must(http.StatusOK, "POST", "/auth/refresh", map[string]any{"refresh_token": stolen})
	fresh, _ := r.obj(t)["refresh_token"].(string)
	if fresh == "" || fresh == stolen {
		t.Fatalf("refresh must rotate the refresh token: %s", r)
	}
	c.must(http.StatusBadRequest, "POST", "/auth/refresh", map[string]any{"refresh_token": stolen})
	c.must(http.StatusBadRequest, "POST", "/auth/refresh", map[string]any{"refresh_token": fresh})

	c.login(password)
	c.must(http.StatusNoContent, "POST", "/auth/logout", map[string]any{"refresh_token": c.refresh})
	c.must(http.StatusUnauthorized, "GET", "/users/me", nil)
	c.must(http.StatusBadRequest, "POST", "/auth/refresh", map[string]any{"refresh_token": c.refresh})
}

func TestSecurity_PasswordBruteforceLocksTheAccount(t *testing.T) {
	c := newUser(t, "secbrute")
	c.token = ""
	for i := range 5 {
		if st := c.do("POST", "/auth/login", map[string]any{"email": c.email, "password": "wrong-guess"}).status; st != http.StatusUnauthorized {
			t.Fatalf("guess %d: want 401, got %d", i+1, st)
		}
	}
	r := c.do("POST", "/auth/login", map[string]any{"email": c.email, "password": password})
	if r.status != http.StatusTooManyRequests || !strings.Contains(string(r.body), "locked") {
		t.Fatalf("after 5 failures the account must be locked, even to the right password: %s", r)
	}
}

func TestSecurity_RateLimitingPerAddress(t *testing.T) {
	c := anon(t)
	c.pinIP = true
	email := uniqueEmail("ratelimit")
	limited := false
	for range 6 {
		if c.do("POST", "/auth/login", map[string]any{"email": email, "password": "x"}).status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("/auth must throttle one address well before 6 quick attempts")
	}

	h := anon(t)
	var mu sync.Mutex
	codes := map[int]int{}
	var wg sync.WaitGroup
	for range 80 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st := h.do("GET", "/health", nil).status
			mu.Lock()
			codes[st]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if codes[http.StatusTooManyRequests] == 0 {
		t.Fatalf("80 simultaneous requests from one address must hit the global limit, got %v", codes)
	}
}

func TestSecurity_EveryActionIsLoggedWithPlatformDeviceAndVersion(t *testing.T) {
	c := newUser(t, "seclog")
	ev := c.createEvent(map[string]any{"name": "logged " + runID})
	c.must(http.StatusNoContent, "POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": tracks[0]})
	tr := c.queueRow(ev, tracks[0])["id"].(string)
	c.must(http.StatusOK, "POST", "/events/"+ev+"/tracks/"+tr+"/vote", map[string]any{})

	type row struct{ user, platform, device, version, resType, resID string }
	find := func(action, resID string) row {
		t.Helper()
		var r row
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			err := db.QueryRow(context.Background(), `
				SELECT coalesce(user_id::text,''), coalesce(platform,''), coalesce(device_model,''),
				       coalesce(app_version,''), coalesce(resource_type,''), coalesce(resource_id,'')
				  FROM audit_logs
				 WHERE action = $1 AND app_version = $2 AND ($3 = '' OR resource_id = $3)
				 ORDER BY id DESC LIMIT 1`, action, "e2e-"+runID, resID).
				Scan(&r.user, &r.platform, &r.device, &r.version, &r.resType, &r.resID)
			if err == nil {
				return r
			}
			time.Sleep(150 * time.Millisecond)
		}
		t.Fatalf("no audit row for %q", action)
		return r
	}

	for _, action := range []struct{ route, resID string }{
		{"POST /events", ""},
		{"POST /events/:id/tracks", ev},
		{"POST /events/:id/tracks/:trackId/vote", tr},
		{"GET /events/:id", ev},
	} {
		r := find(action.route, action.resID)
		if r.user != c.id || r.platform != "android" || r.device != "e2e-pixel-8" || r.version != "e2e-"+runID {
			t.Fatalf("%s: audit row must name the user, platform, device and version, got %+v", action.route, r)
		}
	}

	ghost := uniqueEmail("ghost")
	anon(t).do("POST", "/auth/login", map[string]any{"email": ghost, "password": "x"})
	var n int
	deadline := time.Now().Add(5 * time.Second)
	for n == 0 && time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		_ = db.QueryRow(context.Background(), `
			SELECT count(*) FROM audit_logs
			 WHERE action = 'POST /auth/login' AND user_id IS NULL AND status_code = 401
			   AND platform = 'android' AND device_model = 'e2e-pixel-8' AND app_version = $1`,
			"e2e-"+runID).Scan(&n)
	}
	if n == 0 {
		t.Fatal("an anonymous, failed action must be logged with its platform, device and version too")
	}
}
