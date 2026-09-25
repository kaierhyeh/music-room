//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRampUp_BenchmarkReportStatesTheServerAndTheLoad(t *testing.T) {
	b, err := os.ReadFile("../docs/BENCHMARK.md")
	if err != nil {
		t.Fatalf("the load report must be committed: %v", err)
	}
	doc := strings.ToLower(string(b))
	for _, want := range []string{"cpu", "ram", "k6", "concurrent"} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/BENCHMARK.md must mention %q", want)
		}
	}
	for _, script := range []string{"auth.js", "vote-ws.js", "playlist-ws.js"} {
		if _, err := os.Stat("../benchmarks/" + script); err != nil {
			t.Errorf("benchmark script %s is missing: %v", script, err)
		}
	}
}

func TestRampUp_SimultaneousUsersOnBothServices(t *testing.T) {
	if testing.Short() {
		t.Skip("load smoke skipped in -short mode")
	}
	const users = 30
	host := newUser(t, "loadhost")
	host.premium()
	ev := host.createEvent(map[string]any{"name": "load " + runID})
	pl := host.createPlaylist(map[string]any{"name": "load " + runID})
	for _, id := range tracks[:3] {
		host.must(http.StatusNoContent, "POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": id})
	}
	queue := host.queue(ev)

	crowd := make([]*client, users)
	for i := range crowd {
		crowd[i] = newUser(t, fmt.Sprintf("load%d", i))
		crowd[i].premium()
	}

	var failures atomic.Int32
	var wg sync.WaitGroup
	start := time.Now()
	for i, c := range crowd {
		wg.Add(1)
		go func(i int, c *client) {
			defer wg.Done()
			tr := queue[i%len(queue)]["id"].(string)
			for _, r := range []resp{
				c.do("GET", "/events/"+ev, nil),
				c.do("POST", "/events/"+ev+"/tracks/"+tr+"/vote", map[string]any{}),
				c.do("GET", "/playlists/"+pl, nil),
				c.do("POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[i%len(tracks)]}),
			} {
				if r.status >= 500 || r.status == http.StatusTooManyRequests ||
					(r.status >= 400 && r.status != http.StatusConflict) {
					failures.Add(1)
					t.Errorf("under load: %s", r)
				}
			}
		}(i, c)
	}
	wg.Wait()
	t.Logf("%d users × 4 requests in %s", users, time.Since(start))

	if failures.Load() > 0 {
		t.Fatalf("%d requests failed under load", failures.Load())
	}
	var total float64
	for _, r := range host.queue(ev) {
		total += r["vote_count"].(float64)
	}
	if total != users {
		t.Fatalf("every one of the %d votes must be counted, got %v", users, total)
	}
	if rows, _ := host.playlist(pl); len(rows) != len(tracks) {
		t.Fatalf("the playlist must hold each distinct track once, got %d", len(rows))
	}
}
