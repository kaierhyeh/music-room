//go:build e2e

package e2e

import (
	"net/http"
	"testing"
)

func TestBonus_FreeAndPaidSubscription(t *testing.T) {
	owner := newUser(t, "subowner")
	guest := newUser(t, "subguest")

	if tier := owner.must(http.StatusOK, "GET", "/users/me/subscription", nil).obj(t)["tier"]; tier != "free" {
		t.Fatalf("a new account starts on the free offer, got %v", tier)
	}
	owner.must(http.StatusBadRequest, "PUT", "/users/me/subscription", map[string]any{"tier": "platinum"})

	pl := owner.createPlaylist(map[string]any{"name": "collab " + runID})
	owner.must(http.StatusCreated, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[0]})
	owner.must(http.StatusPaymentRequired, "POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": guest.id, "can_edit": true})
	guest.must(http.StatusPaymentRequired, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[1]})

	owner.premium()
	guest.premium()
	owner.must(http.StatusNoContent, "POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": guest.id, "can_edit": true})
	guest.must(http.StatusCreated, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[1]})

	guest.must(http.StatusOK, "PUT", "/users/me/subscription", map[string]any{"tier": "free"})
	guest.must(http.StatusPaymentRequired, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[2]})
	if h := guest.must(http.StatusOK, "GET", "/users/me/subscription/history", nil).list(t); len(h) != 2 {
		t.Fatalf("both switches must be in the history, got %v", h)
	}
}

func TestBonus_WebClientOriginIsAllowed(t *testing.T) {
	preflight := func(origin string) string {
		t.Helper()
		req, _ := http.NewRequest("OPTIONS", apiURL+"/auth/login", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("X-Real-IP", nextIP())
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("preflight: %v", err)
		}
		res.Body.Close()
		return res.Header.Get("Access-Control-Allow-Origin")
	}
	if got := preflight("http://localhost:8081"); got != "http://localhost:8081" {
		t.Fatalf("the web client's origin must be allowed, got %q", got)
	}
	if got := preflight("https://evil.example"); got != "" {
		t.Fatalf("an unknown origin must not be allowed, got %q", got)
	}
}
