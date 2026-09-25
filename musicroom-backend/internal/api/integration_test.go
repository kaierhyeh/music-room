package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nfauconn/music-room/musicroom-backend/internal/api"
	"github.com/nfauconn/music-room/musicroom-backend/internal/config"
)

type env struct {
	srv  *httptest.Server
	pool *pgxpool.Pool
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — skipping integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	resetSchema(t, pool)

	deezer := httptest.NewServer(http.HandlerFunc(stubDeezer))
	t.Cleanup(deezer.Close)

	cfg := &config.Config{
		JWTSecret:          "integration-test-secret",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      720 * time.Hour,
		DeezerAPIBase:      deezer.URL,
		SMTPFrom:           "noreply@test.local",
		GoogleClientID:     "test-client-id.apps.googleusercontent.com",
		GoogleClientSecret: "test-client-secret",
		GoogleRedirectURL:  "http://localhost:8080/auth/google/callback",
		AppRedirectURL:     "musicroom://auth",
	}

	e, cleanup := api.NewRouter(cfg, pool)
	srv := httptest.NewServer(e)
	t.Cleanup(func() {
		srv.Close()
		cleanup()
		pool.Close()
	})
	return &env{srv: srv, pool: pool}
}

func resetSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	sort.Strings(files)
	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(f), err)
		}
	}
}

func stubDeezer(w http.ResponseWriter, r *http.Request) {
	track := func(id int64) map[string]any {
		return map[string]any{
			"id": id, "title": fmt.Sprintf("Track %d", id), "duration": 210,
			"preview": fmt.Sprintf("https://example.invalid/%d.mp3", id),
			"artist":  map[string]any{"name": "Test Artist"},
			"album":   map[string]any{"title": "Test Album", "cover_medium": ""},
		}
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case strings.HasPrefix(r.URL.Path, "/search"):
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{track(1), track(2), track(3)},
		})
	case strings.HasPrefix(r.URL.Path, "/track/"):
		var id int64
		fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/track/"), "%d", &id)
		_ = json.NewEncoder(w).Encode(track(id))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

var ipCounter struct {
	sync.Mutex
	n int
}

func nextIP() string {
	ipCounter.Lock()
	defer ipCounter.Unlock()
	ipCounter.n++
	return fmt.Sprintf("10.9.%d.%d", ipCounter.n/250+1, ipCounter.n%250+1)
}

type client struct {
	t     *testing.T
	env   *env
	ip    string
	token string
	id    string
}

func (e *env) client(t *testing.T) *client {
	return &client{t: t, env: e, ip: nextIP()}
}

func (c *client) rotateIP() { c.ip = nextIP() }

func (c *client) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, err := http.NewRequest(method, c.env.srv.URL+path, rdr)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", c.ip)
	req.Header.Set("X-Platform", "android")
	req.Header.Set("X-Device-Model", "test-device")
	req.Header.Set("X-App-Version", "1.0.0-test")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	buf := new(strings.Builder)
	_, _ = fmt.Fprint(buf, "")
	data := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := res.Body.Read(tmp)
		data = append(data, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return res.StatusCode, data
}

func (c *client) doNoRedirect(method, path string) (int, string) {
	c.t.Helper()
	req, err := http.NewRequest(method, c.env.srv.URL+path, nil)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Real-IP", c.ip)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	cl := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	res, err := cl.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	return res.StatusCode, res.Header.Get("Location")
}

func (c *client) json(status int, data []byte) map[string]any {
	c.t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		c.t.Fatalf("decode (status %d): %v — body %s", status, err, data)
	}
	return m
}

func (e *env) newUser(t *testing.T, name string) *client {
	t.Helper()
	c := e.client(t)
	email := name + "@test.local"
	if st, body := c.do("POST", "/auth/register", map[string]any{
		"email": email, "password": "password123", "display_name": name,
	}); st != http.StatusCreated {
		t.Fatalf("register %s: %d — %s", name, st, body)
	}
	if _, err := e.pool.Exec(context.Background(),
		"UPDATE users SET email_verified = true WHERE email = $1", email); err != nil {
		t.Fatalf("verify %s: %v", name, err)
	}
	st, body := c.do("POST", "/auth/login", map[string]any{"email": email, "password": "password123"})
	if st != http.StatusOK {
		t.Fatalf("login %s: %d — %s", name, st, body)
	}
	c.token, _ = c.json(st, body)["access_token"].(string)

	st, body = c.do("GET", "/users/me", nil)
	if st != http.StatusOK {
		t.Fatalf("me %s: %d — %s", name, st, body)
	}
	c.id, _ = c.json(st, body)["id"].(string)
	return c
}

func (c *client) createEvent(body map[string]any) string {
	c.t.Helper()
	st, data := c.do("POST", "/events", body)
	if st != http.StatusCreated {
		c.t.Fatalf("create event: %d — %s", st, data)
	}
	id, _ := c.json(st, data)["id"].(string)
	return id
}

func (c *client) queue(eventID string) []any {
	c.t.Helper()
	st, data := c.do("GET", "/events/"+eventID, nil)
	if st != http.StatusOK {
		c.t.Fatalf("get event: %d — %s", st, data)
	}
	q, _ := c.json(st, data)["queue"].([]any)
	return q
}

func (c *client) wsTicket() string {
	c.t.Helper()
	st, data := c.do("POST", "/ws-tickets", nil)
	if st != http.StatusOK {
		c.t.Fatalf("ws-ticket: %d — %s", st, data)
	}
	tk, _ := c.json(st, data)["ticket"].(string)
	return tk
}

func (e *env) dialWS(t *testing.T, path, ticket, origin string) int {
	t.Helper()
	u := "ws" + strings.TrimPrefix(e.srv.URL, "http") + path + "?ticket=" + ticket
	h := http.Header{}
	if origin != "" {
		h.Set("Origin", origin)
	}
	conn, res, err := websocket.DefaultDialer.Dial(u, h)
	if conn != nil {
		defer conn.Close()
	}
	if res != nil {
		return res.StatusCode
	}
	t.Fatalf("dial %s: %v", path, err)
	return 0
}

func TestSuggestTrackPayloadContract(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	ev := alice.createEvent(map[string]any{"name": "contract"})

	if st, body := alice.do("POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusNoContent {
		t.Fatalf("deezer_id should be accepted, got %d — %s", st, body)
	}
	if st, _ := alice.do("POST", "/events/"+ev+"/tracks", map[string]any{"track_deezer_id": 222}); st != http.StatusBadRequest {
		t.Fatalf("track_deezer_id should be rejected, got %d", st)
	}
	if n := len(alice.queue(ev)); n != 1 {
		t.Fatalf("queue should hold exactly the accepted track, got %d", n)
	}
}

func TestQueueIsServedByTheEventPayload(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	ev := alice.createEvent(map[string]any{"name": "queue"})
	if st, _ := alice.do("POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusNoContent {
		t.Fatal("suggest failed")
	}
	if st, _ := alice.do("GET", "/events/"+ev+"/tracks", nil); st != http.StatusNotFound {
		t.Fatalf("GET /events/:id/tracks is not a route; got %d", st)
	}
	if n := len(alice.queue(ev)); n != 1 {
		t.Fatalf("event payload should carry the queue, got %d tracks", n)
	}
}

func TestPlaylistMoveHonoursAfterTrackID(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")

	st, data := alice.do("POST", "/playlists", map[string]any{"name": "moves"})
	if st != http.StatusCreated {
		t.Fatalf("create playlist: %d — %s", st, data)
	}
	pl, _ := alice.json(st, data)["id"].(string)

	for _, id := range []int{111, 222, 333} {
		if st, body := alice.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": id}); st != http.StatusCreated {
			t.Fatalf("add %d: %d — %s", id, st, body)
		}
	}

	read := func() ([]any, int) {
		st, data := alice.do("GET", "/playlists/"+pl, nil)
		m := alice.json(st, data)
		tracks, _ := m["tracks"].([]any)
		v, _ := m["version"].(float64)
		return tracks, int(v)
	}
	tracks, version := read()
	if len(tracks) != 3 {
		t.Fatalf("expected 3 tracks, got %d", len(tracks))
	}
	first := tracks[0].(map[string]any)["id"].(string)
	third := tracks[2].(map[string]any)["id"].(string)

	if st, body := alice.do("PATCH", "/playlists/"+pl+"/tracks/"+third,
		map[string]any{"after_track_id": first, "version": version}); st != http.StatusOK {
		t.Fatalf("move: %d — %s", st, body)
	}

	tracks, _ = read()
	if got := tracks[1].(map[string]any)["id"].(string); got != third {
		t.Fatalf("moved track should sit in position 2, not at the head; order is %v",
			[]string{
				tracks[0].(map[string]any)["id"].(string),
				tracks[1].(map[string]any)["id"].(string),
				tracks[2].(map[string]any)["id"].(string),
			})
	}
}

func TestEmailVerificationGatesLogin(t *testing.T) {
	e := newEnv(t)
	c := e.client(t)
	if st, body := c.do("POST", "/auth/register", map[string]any{
		"email": "unverified@test.local", "password": "password123", "display_name": "unverified",
	}); st != http.StatusCreated {
		t.Fatalf("register: %d — %s", st, body)
	}

	login := func() (int, []byte) {
		c.rotateIP()
		return c.do("POST", "/auth/login", map[string]any{
			"email": "unverified@test.local", "password": "password123",
		})
	}
	st, body := login()
	if st != http.StatusForbidden {
		t.Fatalf("unverified account must not log in, got %d — %s", st, body)
	}
	if msg, _ := c.json(st, body)["message"].(string); msg != "email not verified" {
		t.Fatalf("unexpected message: %q", msg)
	}

	c.rotateIP()
	if st, _ := c.do("POST", "/auth/login", map[string]any{
		"email": "unverified@test.local", "password": "wrong-password",
	}); st != http.StatusUnauthorized {
		t.Fatalf("wrong password should be 401, got %d", st)
	}

	if _, err := e.pool.Exec(context.Background(),
		"UPDATE users SET email_verified = true WHERE email = 'unverified@test.local'"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if st, body := login(); st != http.StatusOK {
		t.Fatalf("verified account must log in, got %d — %s", st, body)
	}
}

func TestWebSocketEnforcesRoomAccess(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	bob := e.newUser(t, "bob")

	private := alice.createEvent(map[string]any{"name": "private", "visibility": "private"})
	public := alice.createEvent(map[string]any{"name": "public"})

	if st, _ := bob.do("GET", "/events/"+private, nil); st != http.StatusForbidden {
		t.Fatalf("REST must refuse bob on a private event, got %d", st)
	}
	if got := e.dialWS(t, "/ws/events/"+private, bob.wsTicket(), ""); got != http.StatusForbidden {
		t.Fatalf("WebSocket must refuse bob on the same event, got %d", got)
	}
	if got := e.dialWS(t, "/ws/events/"+private, alice.wsTicket(), ""); got != http.StatusSwitchingProtocols {
		t.Fatalf("the owner must still get in, got %d", got)
	}
	if got := e.dialWS(t, "/ws/events/"+public, bob.wsTicket(), ""); got != http.StatusSwitchingProtocols {
		t.Fatalf("a public event must stay open, got %d", got)
	}
	if got := e.dialWS(t, "/ws/events/"+public, "not-a-real-ticket", ""); got != http.StatusUnauthorized {
		t.Fatalf("a bogus ticket must be refused, got %d", got)
	}
}

func TestWebSocketRoomsAreTyped(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	ev := alice.createEvent(map[string]any{"name": "typed"})

	st, data := alice.do("POST", "/playlists", map[string]any{"name": "typed"})
	pl, _ := alice.json(st, data)["id"].(string)

	if got := e.dialWS(t, "/ws/playlists/"+ev, alice.wsTicket(), ""); got != http.StatusForbidden {
		t.Fatalf("an event id on the playlist route must be refused, got %d", got)
	}
	if got := e.dialWS(t, "/ws/events/"+pl, alice.wsTicket(), ""); got != http.StatusForbidden {
		t.Fatalf("a playlist id on the event route must be refused, got %d", got)
	}
	if got := e.dialWS(t, "/ws/playlists/"+pl, alice.wsTicket(), ""); got != http.StatusSwitchingProtocols {
		t.Fatalf("the matching route must work, got %d", got)
	}
}

func TestWebSocketAcceptsSameOrigin(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	ev := alice.createEvent(map[string]any{"name": "origins"})

	if got := e.dialWS(t, "/ws/events/"+ev, alice.wsTicket(), e.srv.URL); got != http.StatusSwitchingProtocols {
		t.Fatalf("same-origin must be accepted, got %d", got)
	}
	if got := e.dialWS(t, "/ws/events/"+ev, alice.wsTicket(), ""); got != http.StatusSwitchingProtocols {
		t.Fatalf("an absent Origin must be accepted, got %d", got)
	}
}

func TestWebSocketAuditNamesTheTicketHolder(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	ev := alice.createEvent(map[string]any{"name": "audited socket"})

	if got := e.dialWS(t, "/ws/events/"+ev, alice.wsTicket(), ""); got != http.StatusSwitchingProtocols {
		t.Fatalf("handshake: %d", got)
	}

	var user, platform, device, version string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		err := e.pool.QueryRow(context.Background(),
			`SELECT coalesce(user_id::text,''), coalesce(platform,''), coalesce(device_model,''), coalesce(app_version,'')
			   FROM audit_logs
			  WHERE action = 'GET /ws/events/:id' AND resource_id = $1
			  ORDER BY created_at DESC LIMIT 1`, ev).Scan(&user, &platform, &device, &version)
		if err == nil {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if user != alice.id || platform != "android" || device != "test-device" || version != "1.0.0-test" {
		t.Fatalf("ws audit row should name alice and her client, got user=%q %q %q %q", user, platform, device, version)
	}
}

func TestTimeAndLocationLicence(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	bob := e.newUser(t, "bob")

	geo := map[string]any{
		"vote_latitude": 48.8566, "vote_longitude": 2.3522, "vote_radius_meters": 500,
		"license": "time_and_location",
	}
	mk := func(name string, start, end time.Time) string {
		body := map[string]any{"name": name}
		for k, v := range geo {
			body[k] = v
		}
		body["vote_start_at"] = start.UTC().Format(time.RFC3339)
		body["vote_end_at"] = end.UTC().Format(time.RFC3339)
		return alice.createEvent(body)
	}
	now := time.Now()
	open := mk("open window", now.Add(-time.Hour), now.Add(time.Hour))
	closed := mk("closed window", now.Add(-3*time.Hour), now.Add(-2*time.Hour))

	if st, _ := alice.do("POST", "/events/"+open+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusNoContent {
		t.Fatal("suggesting inside the window must work")
	}
	if st, _ := bob.do("POST", "/events/"+closed+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusForbidden {
		t.Fatalf("suggesting outside the window must be refused, got %d", st)
	}

	track := alice.queue(open)[0].(map[string]any)["id"].(string)
	paris := map[string]any{"latitude": 48.8566, "longitude": 2.3522}
	lyon := map[string]any{"latitude": 45.7640, "longitude": 4.8357}

	if st, _ := bob.do("POST", "/events/"+open+"/tracks/"+track+"/vote", map[string]any{}); st != http.StatusForbidden {
		t.Fatalf("missing coordinates must be 403, got %d", st)
	}
	if st, _ := bob.do("POST", "/events/"+open+"/tracks/"+track+"/vote", lyon); st != http.StatusForbidden {
		t.Fatalf("voting from outside the radius must be 403, got %d", st)
	}
	if st, body := bob.do("POST", "/events/"+open+"/tracks/"+track+"/vote", paris); st != http.StatusOK {
		t.Fatalf("voting inside window and radius must work, got %d — %s", st, body)
	}
	if st, _ := bob.do("DELETE", "/events/"+open+"/tracks/"+track+"/vote", nil); st != http.StatusOK {
		t.Fatal("withdrawing inside the window must work")
	}
}

func TestAuditRecordsTheResourceActedOn(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	ev := alice.createEvent(map[string]any{"name": "audit"})
	if st, _ := alice.do("POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusNoContent {
		t.Fatal("suggest failed")
	}
	track := alice.queue(ev)[0].(map[string]any)["id"].(string)
	if st, _ := alice.do("POST", "/events/"+ev+"/tracks/"+track+"/vote", map[string]any{}); st != http.StatusOK {
		t.Fatal("vote failed")
	}

	var resType, resID string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		err := e.pool.QueryRow(context.Background(),
			`SELECT coalesce(resource_type,''), coalesce(resource_id,'')
			   FROM audit_logs
			  WHERE action = 'POST /events/:id/tracks/:trackId/vote'
			  ORDER BY created_at DESC LIMIT 1`).Scan(&resType, &resID)
		if err == nil && resID != "" {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if resType != "track" || resID != track {
		t.Fatalf("audit row should name the track voted on; got type=%q id=%q (want %q)", resType, resID, track)
	}

	var platform, device, version string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT coalesce(platform,''), coalesce(device_model,''), coalesce(app_version,'')
		   FROM audit_logs WHERE platform <> '' ORDER BY created_at DESC LIMIT 1`).
		Scan(&platform, &device, &version); err != nil {
		t.Fatalf("audit headers: %v", err)
	}
	if platform != "android" || device != "test-device" || version != "1.0.0-test" {
		t.Fatalf("client headers not recorded: %q %q %q", platform, device, version)
	}
}

func TestConcurrentVotes(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	bob := e.newUser(t, "bob")
	carol := e.newUser(t, "carol")

	ev := alice.createEvent(map[string]any{"name": "concurrency"})
	if st, _ := alice.do("POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusNoContent {
		t.Fatal("suggest failed")
	}
	track := alice.queue(ev)[0].(map[string]any)["id"].(string)

	var wg sync.WaitGroup
	for _, c := range []*client{alice, bob, carol} {
		wg.Add(1)
		go func(c *client) {
			defer wg.Done()
			if st, body := c.do("POST", "/events/"+ev+"/tracks/"+track+"/vote", map[string]any{}); st != http.StatusOK {
				t.Errorf("vote: %d — %s", st, body)
			}
		}(c)
	}
	wg.Wait()
	if got := alice.queue(ev)[0].(map[string]any)["vote_count"].(float64); got != 3 {
		t.Fatalf("expected 3 votes, got %v", got)
	}

	dave := e.newUser(t, "dave")
	results := make([]bool, 2)
	wg = sync.WaitGroup{}
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			st, body := dave.do("POST", "/events/"+ev+"/tracks/"+track+"/vote", map[string]any{})
			if st != http.StatusOK {
				t.Errorf("duplicate vote: %d — %s", st, body)
				return
			}
			var m map[string]any
			_ = json.Unmarshal(body, &m)
			results[i], _ = m["voted"].(bool)
		}(i)
	}
	wg.Wait()
	if results[0] == results[1] {
		t.Fatalf("a duplicate vote must report voted=false exactly once, got %v", results)
	}
	if got := alice.queue(ev)[0].(map[string]any)["vote_count"].(float64); got != 4 {
		t.Fatalf("dave should count once and only once, got %v", got)
	}
}

func TestQueueReportsWhetherTheViewerVoted(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	bob := e.newUser(t, "bob")
	ev := alice.createEvent(map[string]any{"name": "voted flag"})
	if st, _ := alice.do("POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": 3135556}); st != http.StatusNoContent {
		t.Fatal("suggest failed")
	}

	voted := func(c *client) bool {
		c.t.Helper()
		row := c.queue(ev)[0].(map[string]any)
		v, ok := row["voted"].(bool)
		if !ok {
			c.t.Fatalf("queue row carries no boolean `voted`: %v", row)
		}
		return v
	}
	track := alice.queue(ev)[0].(map[string]any)["id"].(string)

	if voted(alice) || voted(bob) {
		t.Fatal("nobody has voted yet, both must read false")
	}
	if st, body := alice.do("POST", "/events/"+ev+"/tracks/"+track+"/vote", map[string]any{}); st != http.StatusOK {
		t.Fatalf("vote: %d — %s", st, body)
	}
	if !voted(alice) {
		t.Fatal("alice voted, her own queue must report voted=true")
	}
	if voted(bob) {
		t.Fatal("bob did not vote; the flag must not leak from another user")
	}

	if st, body := alice.do("DELETE", "/events/"+ev+"/tracks/"+track+"/vote", nil); st != http.StatusOK {
		t.Fatalf("remove vote: %d — %s", st, body)
	}
	if voted(alice) {
		t.Fatal("vote withdrawn, the flag must return to false")
	}
}

func TestGoogleLinkTicketAuthorisesTheRoundTrip(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")

	anon := e.client(t)
	if st, body := anon.do("POST", "/users/me/oauth/google/link-ticket", nil); st != http.StatusUnauthorized {
		t.Fatalf("link ticket without a session: %d — %s", st, body)
	}

	st, body := alice.do("POST", "/users/me/oauth/google/link-ticket", nil)
	if st != http.StatusOK {
		t.Fatalf("link ticket: %d — %s", st, body)
	}
	ticket, _ := alice.json(st, body)["ticket"].(string)
	if ticket == "" {
		t.Fatal("no ticket returned")
	}

	code, loc := alice.doNoRedirect("GET",
		"/auth/google/start?redirect=exp%3A%2F%2F192.168.1.2%3A8082%2F--%2Fauth&link="+ticket)
	if code != http.StatusFound {
		t.Fatalf("start with a link ticket: %d", code)
	}
	if !strings.HasPrefix(loc, "https://accounts.google.com/") {
		t.Fatalf("start must send the browser to Google, got %q", loc)
	}

	code, loc = alice.doNoRedirect("GET", "/auth/google/start?link=not-a-ticket")
	if code != http.StatusFound || !strings.HasPrefix(loc, "https://accounts.google.com/") {
		t.Fatalf("an unknown ticket must degrade to a plain sign-in, got %d %q", code, loc)
	}
}

func TestPasswordResetRetiresAccessTokens(t *testing.T) {
	e := newEnv(t)
	victim := e.newUser(t, "resetvictim")
	bystander := e.newUser(t, "resetbystander")

	if st, body := victim.do("GET", "/users/me", nil); st != http.StatusOK {
		t.Fatalf("the token must work before the reset, got %d — %s", st, body)
	}

	token := "reset-token-planted-by-the-test"
	sum := sha256.Sum256([]byte(token))
	if _, err := e.pool.Exec(context.Background(),
		"INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, $2, now() + interval '1 hour')",
		victim.id, sum[:]); err != nil {
		t.Fatalf("plant the reset token: %v", err)
	}

	victim.rotateIP()
	if st, body := victim.do("POST", "/auth/reset-password", map[string]any{
		"token": token, "password": "newpassword456",
	}); st != http.StatusNoContent {
		t.Fatalf("reset: %d — %s", st, body)
	}

	if st, body := victim.do("GET", "/users/me", nil); st != http.StatusUnauthorized {
		t.Fatalf("the access token must not survive the reset, got %d — %s", st, body)
	}
	if st, body := bystander.do("GET", "/users/me", nil); st != http.StatusOK {
		t.Fatalf("another user's session must be untouched, got %d — %s", st, body)
	}

	victim.rotateIP()
	st, body := victim.do("POST", "/auth/login", map[string]any{
		"email": "resetvictim@test.local", "password": "newpassword456",
	})
	if st != http.StatusOK {
		t.Fatalf("login with the new password: %d — %s", st, body)
	}
	victim.token, _ = victim.json(st, body)["access_token"].(string)
	if st, body := victim.do("GET", "/users/me", nil); st != http.StatusOK {
		t.Fatalf("a token minted after the reset must work, got %d — %s", st, body)
	}
}

func TestLogoutEndsOnlyTheCallingSession(t *testing.T) {
	e := newEnv(t)
	c := e.newUser(t, "twodevices")

	login := func() (access, refresh string) {
		t.Helper()
		c.rotateIP()
		st, body := c.do("POST", "/auth/login", map[string]any{
			"email": "twodevices@test.local", "password": "password123",
		})
		if st != http.StatusOK {
			t.Fatalf("login: %d — %s", st, body)
		}
		m := c.json(st, body)
		a, _ := m["access_token"].(string)
		r, _ := m["refresh_token"].(string)
		return a, r
	}
	phoneAccess, phoneRefresh := login()
	laptopAccess, _ := login()

	c.token = phoneAccess
	if st, body := c.do("POST", "/auth/logout", map[string]any{
		"refresh_token": phoneRefresh,
	}); st != http.StatusNoContent {
		t.Fatalf("logout: %d — %s", st, body)
	}

	if st, body := c.do("GET", "/users/me", nil); st != http.StatusUnauthorized {
		t.Fatalf("the access token of the logged-out session must be refused, got %d — %s", st, body)
	}
	c.rotateIP()
	if st, body := c.do("POST", "/auth/refresh", map[string]any{
		"refresh_token": phoneRefresh,
	}); st != http.StatusBadRequest {
		t.Fatalf("the refresh token of the logged-out session must be refused, got %d — %s", st, body)
	}

	c.token = laptopAccess
	if st, body := c.do("GET", "/users/me", nil); st != http.StatusOK {
		t.Fatalf("the other device must stay signed in, got %d — %s", st, body)
	}
}

func TestPremiumGatesPlaylistCollaboration(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(t, "plowner")
	guest := e.newUser(t, "plguest")

	upgrade := func(c *client, tier string) {
		t.Helper()
		if st, body := c.do("PUT", "/users/me/subscription", map[string]any{"tier": tier}); st != http.StatusOK {
			t.Fatalf("switch to %s: %d — %s", tier, st, body)
		}
	}

	st, data := owner.do("POST", "/playlists", map[string]any{"name": "collab", "visibility": "public"})
	if st != http.StatusCreated {
		t.Fatalf("create playlist: %d — %s", st, data)
	}
	pl, _ := owner.json(st, data)["id"].(string)

	if st, body := owner.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusCreated {
		t.Fatalf("a free owner must still edit their own playlist, got %d — %s", st, body)
	}

	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": guest.id}); st != http.StatusPaymentRequired {
		t.Fatalf("a free owner must not add collaborators, got %d — %s", st, body)
	}
	upgrade(owner, "premium")
	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": guest.id}); st != http.StatusNoContent {
		t.Fatalf("a premium owner must add collaborators, got %d — %s", st, body)
	}

	if st, body := guest.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": 222}); st != http.StatusPaymentRequired {
		t.Fatalf("a free collaborator must not edit, got %d — %s", st, body)
	}
	upgrade(guest, "premium")
	if st, body := guest.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": 222}); st != http.StatusCreated {
		t.Fatalf("a premium collaborator must edit, got %d — %s", st, body)
	}

	if st, body := guest.do("GET", "/playlists/"+pl, nil); st != http.StatusOK {
		t.Fatalf("reading a public playlist must stay free, got %d — %s", st, body)
	}
}

func TestPremiumExpiresWithoutAJob(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(t, "expiring")
	other := e.newUser(t, "expiringguest")

	st, data := owner.do("POST", "/playlists", map[string]any{"name": "expiry"})
	if st != http.StatusCreated {
		t.Fatalf("create playlist: %d — %s", st, data)
	}
	pl, _ := owner.json(st, data)["id"].(string)

	if st, body := owner.do("PUT", "/users/me/subscription", map[string]any{"tier": "premium"}); st != http.StatusOK {
		t.Fatalf("upgrade: %d — %s", st, body)
	}
	if st, _ := owner.do("GET", "/users/me/subscription", nil); st != http.StatusOK {
		t.Fatalf("read subscription: %d", st)
	}

	if _, err := e.pool.Exec(context.Background(),
		"UPDATE users SET subscription_expires_at = now() - interval '1 second' WHERE id = $1", owner.id); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	st, data = owner.do("GET", "/users/me/subscription", nil)
	if tier, _ := owner.json(st, data)["tier"].(string); tier != "free" {
		t.Fatalf("a lapsed premium must read as free, got %q", tier)
	}
	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": other.id}); st != http.StatusPaymentRequired {
		t.Fatalf("a lapsed premium must lose the paid feature, got %d — %s", st, body)
	}

	st, data = owner.do("GET", "/users/me/subscription/history", nil)
	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("decode history: %v — %s", err, data)
	}
	if len(entries) != 1 || entries[0]["tier"] != "premium" {
		t.Fatalf("history should hold the upgrade, got %v", entries)
	}
}

func TestPlaylistOwnershipAndMetadataEditing(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(t, "plmeta")
	other := e.newUser(t, "plother")

	st, data := owner.do("POST", "/playlists", map[string]any{"name": "public one", "visibility": "public"})
	if st != http.StatusCreated {
		t.Fatalf("create public playlist: %d — %s", st, data)
	}
	created := owner.json(st, data)
	pub, _ := created["id"].(string)
	if got, _ := created["owner_display_name"].(string); got != "plmeta" {
		t.Fatalf("created playlist must name its owner, got %q", got)
	}

	st, data = owner.do("POST", "/playlists", map[string]any{"name": "private one", "visibility": "private"})
	if st != http.StatusCreated {
		t.Fatalf("create private playlist: %d — %s", st, data)
	}
	priv, _ := owner.json(st, data)["id"].(string)

	names := func(c *client, path string) map[string]string {
		t.Helper()
		st, data := c.do("GET", path, nil)
		if st != http.StatusOK {
			t.Fatalf("GET %s: %d — %s", path, st, data)
		}
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("decode %s: %v — %s", path, err, data)
		}
		out := map[string]string{}
		for _, r := range rows {
			id, _ := r["id"].(string)
			owner, _ := r["owner_display_name"].(string)
			out[id] = owner
		}
		return out
	}

	browse := names(other, "/playlists")
	if _, ok := browse[pub]; !ok {
		t.Fatalf("a public playlist must appear in the browse list")
	}
	if browse[pub] != "plmeta" {
		t.Fatalf("the browse list must name each creator, got %q", browse[pub])
	}
	if _, ok := browse[priv]; ok {
		t.Fatalf("a private playlist must never appear in the browse list")
	}

	mine := names(owner, "/playlists/mine")
	if _, ok := mine[pub]; !ok {
		t.Fatalf("my playlists must include my public playlist")
	}
	if _, ok := mine[priv]; !ok {
		t.Fatalf("my playlists must include my private playlist")
	}
	if others := names(other, "/playlists/mine"); len(others) != 0 {
		t.Fatalf("my playlists is the caller's own, got %d rows for someone else", len(others))
	}

	st, data = owner.do("PATCH", "/playlists/"+priv, map[string]any{"name": "renamed"})
	if st != http.StatusOK {
		t.Fatalf("owner rename: %d — %s", st, data)
	}
	updated := owner.json(st, data)
	if got, _ := updated["name"].(string); got != "renamed" {
		t.Fatalf("name must change, got %q", got)
	}
	if got, _ := updated["visibility"].(string); got != "private" {
		t.Fatalf("an omitted field must keep its value, visibility is now %q", got)
	}

	if st, body := other.do("PATCH", "/playlists/"+pub, map[string]any{"name": "hijacked"}); st != http.StatusForbidden {
		t.Fatalf("a non-owner must not rename a playlist, got %d — %s", st, body)
	}

	if st, body := owner.do("PATCH", "/playlists/"+pub, map[string]any{"visibility": "private"}); st != http.StatusOK {
		t.Fatalf("owner turns playlist private: %d — %s", st, body)
	}
	if _, ok := names(other, "/playlists")[pub]; ok {
		t.Fatalf("a playlist turned private must leave the browse list")
	}
	if st, body := other.do("GET", "/playlists/"+pub, nil); st != http.StatusForbidden {
		t.Fatalf("a playlist turned private must be closed to others, got %d — %s", st, body)
	}
}

func TestPlaylistCollaboratorRemoval(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(t, "plrevoke")
	guest := e.newUser(t, "plrevoked")

	st, data := owner.do("POST", "/playlists", map[string]any{"name": "revocable", "license": "invited_only"})
	if st != http.StatusCreated {
		t.Fatalf("create playlist: %d — %s", st, data)
	}
	pl, _ := owner.json(st, data)["id"].(string)

	if st, body := owner.do("PUT", "/users/me/subscription", map[string]any{"tier": "premium"}); st != http.StatusOK {
		t.Fatalf("upgrade: %d — %s", st, body)
	}
	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": guest.id}); st != http.StatusNoContent {
		t.Fatalf("add collaborator: %d — %s", st, body)
	}

	if st, body := guest.do("PATCH", "/playlists/"+pl, map[string]any{"name": "mine now"}); st != http.StatusForbidden {
		t.Fatalf("a collaborator must not rename the playlist, got %d — %s", st, body)
	}

	if st, body := guest.do("DELETE", "/playlists/"+pl+"/collaborators/"+owner.id, nil); st != http.StatusForbidden {
		t.Fatalf("only the owner revokes an invitation, got %d — %s", st, body)
	}

	if st, body := owner.do("PUT", "/users/me/subscription", map[string]any{"tier": "free"}); st != http.StatusOK {
		t.Fatalf("downgrade: %d — %s", st, body)
	}
	if st, body := owner.do("DELETE", "/playlists/"+pl+"/collaborators/"+guest.id, nil); st != http.StatusNoContent {
		t.Fatalf("a free owner must still revoke, got %d — %s", st, body)
	}

	st, data = owner.do("GET", "/playlists/"+pl+"/collaborators", nil)
	if st != http.StatusOK {
		t.Fatalf("list collaborators: %d — %s", st, data)
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("decode collaborators: %v — %s", err, data)
	}
	if len(rows) != 0 {
		t.Fatalf("the revoked collaborator must be gone, got %v", rows)
	}

	if st, body := owner.do("DELETE", "/playlists/"+pl+"/collaborators/"+guest.id, nil); st != http.StatusNotFound {
		t.Fatalf("revoking a non-collaborator must 404, got %d — %s", st, body)
	}
}

func TestPlaylistViewerIsReadOnly(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(t, "plviewown")
	viewer := e.newUser(t, "plviewer")
	stranger := e.newUser(t, "plviewstr")

	st, data := owner.do("POST", "/playlists", map[string]any{"name": "shown", "visibility": "private", "license": "open"})
	if st != http.StatusCreated {
		t.Fatalf("create playlist: %d — %s", st, data)
	}
	pl, _ := owner.json(st, data)["id"].(string)

	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": viewer.id, "can_edit": true}); st != http.StatusPaymentRequired {
		t.Fatalf("a free owner must not invite an editor, got %d — %s", st, body)
	}
	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": viewer.id, "can_edit": false}); st != http.StatusNoContent {
		t.Fatalf("a free owner must invite a viewer, got %d — %s", st, body)
	}

	if st, body := viewer.do("GET", "/playlists/"+pl, nil); st != http.StatusOK {
		t.Fatalf("a viewer must open the private playlist, got %d — %s", st, body)
	}
	if st, body := stranger.do("GET", "/playlists/"+pl, nil); st != http.StatusForbidden {
		t.Fatalf("a stranger must not open the private playlist, got %d — %s", st, body)
	}

	if st, body := viewer.do("PUT", "/users/me/subscription", map[string]any{"tier": "premium"}); st != http.StatusOK {
		t.Fatalf("upgrade viewer: %d — %s", st, body)
	}
	if st, body := viewer.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusForbidden {
		t.Fatalf("a viewer must not edit, even premium under an open license, got %d — %s", st, body)
	}

	if st, body := owner.do("PUT", "/users/me/subscription", map[string]any{"tier": "premium"}); st != http.StatusOK {
		t.Fatalf("upgrade owner: %d — %s", st, body)
	}
	if st, body := owner.do("POST", "/playlists/"+pl+"/collaborators", map[string]any{"user_id": viewer.id, "can_edit": true}); st != http.StatusNoContent {
		t.Fatalf("promote viewer: %d — %s", st, body)
	}
	if st, body := viewer.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": 111}); st != http.StatusCreated {
		t.Fatalf("a promoted viewer must edit, got %d — %s", st, body)
	}

	st, data = owner.do("GET", "/playlists/"+pl+"/collaborators", nil)
	if st != http.StatusOK {
		t.Fatalf("list collaborators: %d — %s", st, data)
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("decode collaborators: %v — %s", err, data)
	}
	if len(rows) != 1 || rows[0]["can_edit"] != true {
		t.Fatalf("the list must carry the updated right, got %v", rows)
	}
}

func TestPlaylistCreateWithInvites(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(t, "plcrown")
	viewer := e.newUser(t, "plcrview")
	editor := e.newUser(t, "plcredit")

	st, data := owner.do("POST", "/playlists", map[string]any{
		"name": "denied", "visibility": "private",
		"collaborators": []map[string]any{{"user_id": editor.id, "can_edit": true}},
	})
	if st != http.StatusPaymentRequired {
		t.Fatalf("a free owner must not create with an editor, got %d — %s", st, data)
	}
	st, data = owner.do("GET", "/playlists/mine", nil)
	var mine []map[string]any
	if err := json.Unmarshal(data, &mine); err != nil || st != http.StatusOK {
		t.Fatalf("list mine: %d — %s", st, data)
	}
	if len(mine) != 0 {
		t.Fatalf("a refused creation must leave no playlist behind, got %v", mine)
	}

	if st, body := owner.do("PUT", "/users/me/subscription", map[string]any{"tier": "premium"}); st != http.StatusOK {
		t.Fatalf("upgrade: %d — %s", st, body)
	}
	st, data = owner.do("POST", "/playlists", map[string]any{
		"name": "shared", "visibility": "private",
		"collaborators": []map[string]any{
			{"user_id": viewer.id, "can_edit": false},
			{"user_id": editor.id, "can_edit": true},
		},
	})
	if st != http.StatusCreated {
		t.Fatalf("create with invites: %d — %s", st, data)
	}
	pl, _ := owner.json(st, data)["id"].(string)

	for _, tc := range []struct {
		c       *client
		canEdit bool
	}{{viewer, false}, {editor, true}} {
		st, data := tc.c.do("GET", "/playlists/shared", nil)
		if st != http.StatusOK {
			t.Fatalf("list shared: %d — %s", st, data)
		}
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("decode shared: %v — %s", err, data)
		}
		if len(rows) != 1 || rows[0]["id"] != pl || rows[0]["can_edit"] != tc.canEdit {
			t.Fatalf("the invitee must find the private playlist with can_edit=%v, got %v", tc.canEdit, rows)
		}
	}

	st, data = owner.do("GET", "/playlists/shared", nil)
	if st != http.StatusOK || strings.TrimSpace(string(data)) != "[]" {
		t.Fatalf("the owner's shared list must be empty, got %d — %s", st, data)
	}
}

func TestEventListAndDetailSayWhoWhereAndWhen(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	bob := e.newUser(t, "bob")
	carol := e.newUser(t, "carol")

	now := time.Now()
	secret := alice.createEvent(map[string]any{
		"name": "secret", "visibility": "private", "license": "time_and_location",
		"vote_latitude": 48.8566, "vote_longitude": 2.3522, "vote_radius_meters": 500,
		"vote_start_at": now.UTC().Format(time.RFC3339),
		"vote_end_at":   now.Add(time.Hour).UTC().Format(time.RFC3339),
	})
	if st, body := alice.do("POST", "/events/"+secret+"/invitations", map[string]any{"user_id": bob.id}); st != http.StatusNoContent {
		t.Fatalf("invite: %d — %s", st, body)
	}

	listed := func(c *client) bool {
		st, data := c.do("GET", "/events", nil)
		if st != http.StatusOK {
			t.Fatalf("list: %d — %s", st, data)
		}
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r["id"] == secret {
				if r["owner_display_name"] != "alice" {
					t.Fatalf("list row must name the host, got %v", r["owner_display_name"])
				}
				return true
			}
		}
		return false
	}
	if !listed(alice) {
		t.Fatal("the host must see their own private event")
	}
	if !listed(bob) {
		t.Fatal("an invited guest must see the private event")
	}
	if listed(carol) {
		t.Fatal("a stranger must not see a private event")
	}

	st, data := bob.do("GET", "/events/"+secret, nil)
	if st != http.StatusOK {
		t.Fatalf("get: %d — %s", st, data)
	}
	ev := bob.json(st, data)
	if ev["owner_display_name"] != "alice" {
		t.Fatalf("detail must name the host, got %v", ev["owner_display_name"])
	}
	for _, k := range []string{"vote_start_at", "vote_end_at", "vote_latitude", "vote_longitude", "vote_radius_meters"} {
		if ev[k] == nil {
			t.Fatalf("detail must carry %s", k)
		}
	}
}

func TestTrackPreviewIsFetchedFresh(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser(t, "alice")
	st, data := alice.do("GET", "/tracks/42/preview", nil)
	if st != http.StatusOK {
		t.Fatalf("preview: %d — %s", st, data)
	}
	if got := alice.json(st, data)["preview_url"]; got != "https://example.invalid/42.mp3" {
		t.Fatalf("preview_url = %v", got)
	}
}
