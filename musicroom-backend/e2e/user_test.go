//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestUser_MailPasswordSignupDemandsMailValidation(t *testing.T) {
	c := register(t, "signup")

	r := c.do("POST", "/auth/login", map[string]any{"email": c.email, "password": password})
	if r.status != http.StatusForbidden || !strings.Contains(string(r.body), "email not verified") {
		t.Fatalf("an unverified account must not log in, got %s", r)
	}

	c.must(http.StatusConflict, "POST", "/auth/register", map[string]any{
		"email": c.email, "password": password, "display_name": "dup",
	})

	c.must(http.StatusBadRequest, "POST", "/auth/verify-email", map[string]any{"token": "not-the-token"})
	token := mailToken(t, c.email, "verify")
	c.must(http.StatusNoContent, "POST", "/auth/verify-email", map[string]any{"token": token})
	c.must(http.StatusBadRequest, "POST", "/auth/verify-email", map[string]any{"token": token})

	c.login(password)
	me := c.must(http.StatusOK, "GET", "/users/me", nil).obj(t)
	if me["email"] != c.email || me["email_verified"] != true {
		t.Fatalf("the verified account must read back as such, got %v", me)
	}
}

func TestUser_ForgottenPasswordIsResetThroughTheMail(t *testing.T) {
	c := newUser(t, "forgot")
	oldToken := c.token

	c.must(http.StatusAccepted, "POST", "/auth/forgot-password", map[string]any{"email": c.email})
	c.must(http.StatusAccepted, "POST", "/auth/forgot-password", map[string]any{"email": uniqueEmail("nobody")})

	token := mailToken(t, c.email, "reset")
	const newPassword = "e2e-Changed456!"
	c.must(http.StatusNoContent, "POST", "/auth/reset-password", map[string]any{"token": token, "password": newPassword})
	c.must(http.StatusBadRequest, "POST", "/auth/reset-password", map[string]any{"token": token, "password": "e2e-Again789!"})

	c.must(http.StatusUnauthorized, "POST", "/auth/login", map[string]any{"email": c.email, "password": password})
	c.token = oldToken
	c.must(http.StatusUnauthorized, "GET", "/users/me", nil)

	c.login(newPassword)
	c.must(http.StatusOK, "GET", "/users/me", nil)
}

func TestUser_GoogleSignInAndLinking(t *testing.T) {
	c := newUser(t, "google")

	nr := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	start := func(query string) (int, string) {
		t.Helper()
		req, _ := http.NewRequest("GET", apiURL+"/auth/google/start"+query, nil)
		req.Header.Set("X-Real-IP", nextIP())
		res, err := nr.Do(req)
		if err != nil {
			t.Fatalf("start: %v", err)
		}
		res.Body.Close()
		return res.StatusCode, res.Header.Get("Location")
	}

	st, _ := start("")
	if st == http.StatusServiceUnavailable {
		t.Skip("GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET are not set on this stack")
	}
	if st, loc := start(""); st != http.StatusFound || !strings.HasPrefix(loc, "https://accounts.google.com/") {
		t.Fatalf("sign-in must send the browser to Google, got %d %q", st, loc)
	}

	anon(t).must(http.StatusUnauthorized, "POST", "/users/me/oauth/google/link-ticket", nil)
	tk, _ := c.must(http.StatusOK, "POST", "/users/me/oauth/google/link-ticket", nil).obj(t)["ticket"].(string)
	if st, loc := start("?link=" + tk); st != http.StatusFound || !strings.HasPrefix(loc, "https://accounts.google.com/") {
		t.Fatalf("linking must send the browser to Google, got %d %q", st, loc)
	}

	a := anon(t)
	if r := a.do("POST", "/auth/google", map[string]any{"id_token": "forged.id.token"}); r.status == http.StatusOK {
		t.Fatalf("a forged id_token must not sign anyone in: %s", r)
	}
	if r := c.do("POST", "/users/me/oauth/google", map[string]any{"id_token": "forged.id.token"}); r.status < 400 {
		t.Fatalf("a forged id_token must not be linked: %s", r)
	}
	if me := c.must(http.StatusOK, "GET", "/users/me", nil).obj(t); me["google_linked"] != false {
		t.Fatalf("nothing was linked, google_linked must be false: %v", me)
	}
}

func TestUser_ProfileVisibilityLevelsAndMusicPreferences(t *testing.T) {
	owner := newUser(t, "profile")
	friend := newUser(t, "profilefriend")
	stranger := newUser(t, "profilestranger")
	befriend(t, owner, friend)

	updated := owner.must(http.StatusOK, "PATCH", "/users/me", map[string]any{
		"profile_data": map[string]any{
			"bio":      "public bio",
			"location": "friends-only city",
			"age":      33,
		},
		"profile_visibility": map[string]string{
			"bio": "public", "location": "friends", "age": "private", "music_preferences": "friends",
		},
		"music_preferences": map[string]any{"genres": []string{"jazz", "house"}, "artists": []string{"Bonobo"}},
	}).obj(t)
	if pd, _ := updated["profile_data"].(map[string]any); pd["age"] != float64(33) {
		t.Fatalf("the update must come back in full to its owner, got %v", updated)
	}

	view := func(c *client) (map[string]any, bool) {
		t.Helper()
		m := c.must(http.StatusOK, "GET", "/users/"+owner.id, nil).obj(t)
		pd, _ := m["profile_data"].(map[string]any)
		_, prefs := m["music_preferences"]
		if _, leaked := m["email"]; leaked {
			t.Fatalf("an account detail leaked to another user: %v", m)
		}
		return pd, prefs
	}

	pd, prefs := view(friend)
	if pd["bio"] != "public bio" || pd["location"] != "friends-only city" || pd["age"] != nil || !prefs {
		t.Fatalf("a friend sees public + friends fields and the preferences, never private ones; got %v prefs=%v", pd, prefs)
	}
	pd, prefs = view(stranger)
	if pd["bio"] != "public bio" || pd["location"] != nil || pd["age"] != nil || prefs {
		t.Fatalf("a stranger sees public fields only; got %v prefs=%v", pd, prefs)
	}

	owner.must(http.StatusOK, "PATCH", "/users/me", map[string]any{
		"music_preferences": map[string]any{"genres": []string{"techno"}},
	})
	me := owner.must(http.StatusOK, "GET", "/users/me", nil).obj(t)
	raw, _ := json.Marshal(me["music_preferences"])
	if !strings.Contains(string(raw), "techno") || me["profile_data"].(map[string]any)["bio"] != "public bio" {
		t.Fatalf("a partial update must change only what it names, got %v", me)
	}

	owner.must(http.StatusBadRequest, "PATCH", "/users/me", map[string]any{
		"profile_visibility": map[string]string{"bio": "everyone"},
	})
}

func TestUser_FriendRequests(t *testing.T) {
	a := newUser(t, "fra")
	b := newUser(t, "frb")
	c := newUser(t, "frc")

	a.must(http.StatusBadRequest, "POST", "/friends/requests", map[string]any{"addressee_id": a.id})
	a.must(http.StatusCreated, "POST", "/friends/requests", map[string]any{"addressee_id": b.id})
	a.must(http.StatusConflict, "POST", "/friends/requests", map[string]any{"addressee_id": b.id})

	inbox := b.must(http.StatusOK, "GET", "/friends/requests", nil).list(t)
	if len(inbox) != 1 || inbox[0]["requester_id"] != a.id {
		t.Fatalf("b's inbox must hold a's request, got %v", inbox)
	}
	reqID := inbox[0]["id"].(string)

	c.must(http.StatusForbidden, "POST", "/friends/requests/"+reqID+"/accept", nil)
	b.must(http.StatusNoContent, "POST", "/friends/requests/"+reqID+"/accept", nil)

	for _, pair := range [][2]*client{{a, b}, {b, a}} {
		if !contains(pair[0].must(http.StatusOK, "GET", "/friends", nil).list(t), pair[1].id) {
			t.Fatalf("%s and %s must list each other as friends", pair[0].name, pair[1].name)
		}
	}

	c.must(http.StatusCreated, "POST", "/friends/requests", map[string]any{"addressee_id": a.id})
	in := a.must(http.StatusOK, "GET", "/friends/requests", nil).list(t)
	a.must(http.StatusNoContent, "POST", "/friends/requests/"+in[0]["id"].(string)+"/reject", nil)
	if contains(a.must(http.StatusOK, "GET", "/friends", nil).list(t), c.id) {
		t.Fatal("a rejected request must not make friends")
	}
}
