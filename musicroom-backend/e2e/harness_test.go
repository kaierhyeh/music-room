//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	emailDomain = "e2e.musicroom.test"
	password    = "e2e-Password123!"
)

var tracks = []int64{3129775, 3135553, 5936379, 486399342, 135541760, 2582922122, 849355752, 138546809}

var (
	apiURL     = envOr("E2E_API_URL", "http://localhost:8080")
	mailhogURL = envOr("E2E_MAILHOG_URL", "http://localhost:8025")
	dbURL      = envOr("E2E_DATABASE_URL", "postgres://musicroom:musicroom@localhost:5432/musicroom?sslmode=disable")

	runID = fmt.Sprintf("%x", time.Now().UnixNano()&0xffffffff)

	db *pgxpool.Pool
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func TestMain(m *testing.M) {
	res, err := http.Get(apiURL + "/health")
	if err != nil || res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "e2e: API not reachable at %s (start it with `make docker-up`): %v\n", apiURL, err)
		os.Exit(1)
	}
	res.Body.Close()

	db, err = pgxpool.New(context.Background(), dbURL)
	if err == nil {
		err = db.Ping(context.Background())
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: database not reachable at %s: %v\n", dbURL, err)
		os.Exit(1)
	}

	code := m.Run()

	if os.Getenv("E2E_KEEP_DATA") == "" {
		if _, err := db.Exec(context.Background(),
			"DELETE FROM users WHERE email LIKE '%@"+emailDomain+"'"); err != nil {
			fmt.Fprintf(os.Stderr, "e2e: cleanup failed: %v\n", err)
		}
	}
	db.Close()
	os.Exit(code)
}

var ipGen = struct {
	sync.Mutex
	a, n int
}{a: rand.IntN(200) + 20}

func nextIP() string {
	ipGen.Lock()
	defer ipGen.Unlock()
	ipGen.n++
	return fmt.Sprintf("10.%d.%d.%d", ipGen.a, ipGen.n/250%250, ipGen.n%250+1)
}

type client struct {
	t       *testing.T
	name    string
	email   string
	id      string
	token   string
	refresh string
	ip      string
	pinIP   bool
}

type resp struct {
	status int
	body   []byte
	header http.Header
}

func (r resp) obj(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.body, &m); err != nil {
		t.Fatalf("decode object (status %d): %v — %s", r.status, err, r.body)
	}
	return m
}

func (r resp) list(t *testing.T) []map[string]any {
	t.Helper()
	var l []map[string]any
	if err := json.Unmarshal(r.body, &l); err != nil {
		t.Fatalf("decode list (status %d): %v — %s", r.status, err, r.body)
	}
	return l
}

func (r resp) String() string { return fmt.Sprintf("%d — %s", r.status, bytes.TrimSpace(r.body)) }

func anon(t *testing.T) *client { return &client{t: t, ip: nextIP()} }

func (c *client) do(method, path string, body any) resp {
	c.t.Helper()
	return c.doHeaders(method, path, body, nil)
}

func (c *client) doHeaders(method, path string, body any, extra http.Header) resp {
	c.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, apiURL+path, rdr)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Platform", "android")
	req.Header.Set("X-Device-Model", "e2e-pixel-8")
	req.Header.Set("X-App-Version", "e2e-"+runID)
	ip := c.ip
	if strings.HasPrefix(path, "/auth/") && !c.pinIP {
		ip = nextIP()
	}
	req.Header.Set("X-Real-IP", ip)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, vs := range extra {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, body: data, header: res.Header}
}

func (c *client) must(want int, method, path string, body any) resp {
	c.t.Helper()
	r := c.do(method, path, body)
	if r.status != want {
		c.t.Fatalf("%s %s: want %d, got %s", method, path, want, r)
	}
	return r
}

func uniqueEmail(name string) string {
	return fmt.Sprintf("%s-%s-%d@%s", name, runID, rand.IntN(1_000_000), emailDomain)
}

func register(t *testing.T, name string) *client {
	t.Helper()
	c := anon(t)
	c.name, c.email = name, uniqueEmail(name)
	c.must(http.StatusCreated, "POST", "/auth/register", map[string]any{
		"email": c.email, "password": password, "display_name": name,
	})
	return c
}

func newUser(t *testing.T, name string) *client {
	t.Helper()
	c := register(t, name)
	token := mailToken(t, c.email, "verify")
	c.must(http.StatusNoContent, "POST", "/auth/verify-email", map[string]any{"token": token})
	c.login(password)
	me := c.must(http.StatusOK, "GET", "/users/me", nil).obj(t)
	c.id, _ = me["id"].(string)
	return c
}

func (c *client) login(pw string) {
	c.t.Helper()
	r := c.must(http.StatusOK, "POST", "/auth/login", map[string]any{"email": c.email, "password": pw})
	m := r.obj(c.t)
	c.token, _ = m["access_token"].(string)
	c.refresh, _ = m["refresh_token"].(string)
	if c.token == "" || c.refresh == "" {
		c.t.Fatalf("login returned no token pair: %s", r)
	}
}

func (c *client) premium() {
	c.t.Helper()
	c.must(http.StatusOK, "PUT", "/users/me/subscription", map[string]any{"tier": "premium"})
}

func befriend(t *testing.T, a, b *client) {
	t.Helper()
	a.must(http.StatusCreated, "POST", "/friends/requests", map[string]any{"addressee_id": b.id})
	for _, r := range b.must(http.StatusOK, "GET", "/friends/requests", nil).list(t) {
		if r["requester_id"] == a.id {
			b.must(http.StatusNoContent, "POST", fmt.Sprintf("/friends/requests/%s/accept", r["id"]), nil)
			return
		}
	}
	t.Fatalf("%s's request never reached %s's inbox", a.name, b.name)
}

var tokenRe = regexp.MustCompile(`token:\s*\r?\n\s*([A-Za-z0-9_-]{43})`)

func mailToken(t *testing.T, to, subject string) string {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		res, err := http.Get(mailhogURL + "/api/v2/search?kind=to&query=" + url.QueryEscape(to))
		if err != nil {
			t.Fatalf("MailHog not reachable at %s: %v", mailhogURL, err)
		}
		var out struct {
			Items []struct {
				Content struct {
					Headers map[string][]string
					Body    string
				}
			}
		}
		err = json.NewDecoder(res.Body).Decode(&out)
		res.Body.Close()
		if err != nil {
			t.Fatalf("decode MailHog: %v", err)
		}
		for _, it := range out.Items {
			subj := strings.Join(it.Content.Headers["Subject"], " ")
			if !strings.Contains(strings.ToLower(subj), subject) {
				continue
			}
			if m := tokenRe.FindStringSubmatch(it.Content.Body); m != nil {
				return m[1]
			}
			t.Fatalf("mail %q to %s carries no token:\n%s", subj, to, it.Content.Body)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("no %q mail reached %s within 15s", subject, to)
	return ""
}

type wsConn struct {
	t    *testing.T
	conn *websocket.Conn
}

func wsURL(path string) string {
	return "ws" + strings.TrimPrefix(apiURL, "http") + path
}

func (c *client) ticket() string {
	c.t.Helper()
	tk, _ := c.must(http.StatusOK, "POST", "/ws-tickets", nil).obj(c.t)["ticket"].(string)
	return tk
}

func (c *client) dial(path string) (*wsConn, int) {
	c.t.Helper()
	conn, res, err := websocket.DefaultDialer.Dial(wsURL(path)+"?ticket="+c.ticket(), nil)
	if res == nil {
		c.t.Fatalf("dial %s: %v", path, err)
	}
	if conn == nil {
		return nil, res.StatusCode
	}
	c.t.Cleanup(func() { conn.Close() })
	return &wsConn{t: c.t, conn: conn}, res.StatusCode
}

func (w *wsConn) next(typ string) map[string]any {
	w.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		_ = w.conn.SetReadDeadline(deadline)
		_, data, err := w.conn.ReadMessage()
		if err != nil {
			w.t.Fatalf("waiting for %q: %v", typ, err)
		}
		var m map[string]any
		if json.Unmarshal(data, &m) == nil && m["type"] == typ {
			return m
		}
	}
}

func (c *client) createEvent(body map[string]any) string {
	c.t.Helper()
	id, _ := c.must(http.StatusCreated, "POST", "/events", body).obj(c.t)["id"].(string)
	return id
}

func (c *client) event(id string) map[string]any {
	c.t.Helper()
	return c.must(http.StatusOK, "GET", "/events/"+id, nil).obj(c.t)
}

func (c *client) queue(eventID string) []map[string]any {
	c.t.Helper()
	raw, _ := c.event(eventID)["queue"].([]any)
	out := make([]map[string]any, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]any)
	}
	return out
}

func (c *client) queueRow(eventID string, deezerID int64) map[string]any {
	c.t.Helper()
	for _, r := range c.queue(eventID) {
		if int64(r["track_deezer_id"].(float64)) == deezerID {
			return r
		}
	}
	c.t.Fatalf("track %d is not in the queue of %s", deezerID, eventID)
	return nil
}

func (c *client) createPlaylist(body map[string]any) string {
	c.t.Helper()
	id, _ := c.must(http.StatusCreated, "POST", "/playlists", body).obj(c.t)["id"].(string)
	return id
}

func (c *client) playlist(id string) ([]map[string]any, int) {
	c.t.Helper()
	m := c.must(http.StatusOK, "GET", "/playlists/"+id, nil).obj(c.t)
	raw, _ := m["tracks"].([]any)
	out := make([]map[string]any, len(raw))
	for i, r := range raw {
		out[i] = r.(map[string]any)
	}
	v, _ := m["version"].(float64)
	return out, int(v)
}

func deezerIDs(rows []map[string]any) []int64 {
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = int64(r["track_deezer_id"].(float64))
	}
	return out
}

func contains(rows []map[string]any, id string) bool {
	for _, r := range rows {
		if r["id"] == id {
			return true
		}
	}
	return false
}

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }
