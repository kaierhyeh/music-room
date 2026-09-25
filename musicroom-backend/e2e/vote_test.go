//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestVote_SuggestVoteAndTheQueueReordersLive(t *testing.T) {
	host := newUser(t, "vhost")
	guests := []*client{newUser(t, "vg1"), newUser(t, "vg2")}

	ev := host.createEvent(map[string]any{"name": "party " + runID})
	room, st := guests[0].dial("/ws/events/" + ev)
	if st != http.StatusSwitchingProtocols {
		t.Fatalf("a guest must join the live room of a public event, got %d", st)
	}

	first, second := tracks[0], tracks[1]
	host.must(http.StatusNoContent, "POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": first})
	room.next("queue:updated")
	guests[1].must(http.StatusNoContent, "POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": second})
	room.next("queue:updated")

	target := host.queueRow(ev, second)["id"].(string)
	for _, g := range guests {
		if v := g.must(http.StatusOK, "POST", "/events/"+ev+"/tracks/"+target+"/vote", map[string]any{}).obj(t); v["voted"] != true {
			t.Fatalf("a first vote must count: %v", v)
		}
	}

	var pushed []any
	for {
		msg := room.next("queue:updated")
		pushed, _ = msg["queue"].([]any)
		if len(pushed) == 2 && pushed[0].(map[string]any)["vote_count"] == float64(2) {
			break
		}
	}
	if got := int64(pushed[0].(map[string]any)["track_deezer_id"].(float64)); got != second {
		t.Fatalf("the pushed queue must put the most voted track first, got %d", got)
	}
	if got := deezerIDs(host.queue(ev)); got[0] != second || got[1] != first {
		t.Fatalf("the most voted track must play first, queue is %v", got)
	}

	if v := guests[0].must(http.StatusOK, "POST", "/events/"+ev+"/tracks/"+target+"/vote", map[string]any{}).obj(t); v["voted"] != false {
		t.Fatalf("a repeated vote must not count again: %v", v)
	}
	guests[0].must(http.StatusOK, "DELETE", "/events/"+ev+"/tracks/"+target+"/vote", nil)
	if n := host.queueRow(ev, second)["vote_count"]; n != float64(1) {
		t.Fatalf("a withdrawn vote must leave 1, got %v", n)
	}
}

func TestVote_PublicByDefaultPrivateForInvitedOnly(t *testing.T) {
	host := newUser(t, "vishost")
	invited := newUser(t, "visinvited")
	outsider := newUser(t, "visoutsider")

	listed := func(c *client, id string) bool {
		t.Helper()
		return contains(c.must(http.StatusOK, "GET", "/events", nil).list(t), id)
	}

	pub := host.createEvent(map[string]any{"name": "default " + runID})
	if v := host.event(pub)["visibility"]; v != "public" {
		t.Fatalf("an event must be public by default, got %v", v)
	}
	if !listed(outsider, pub) {
		t.Fatal("anyone must find a public event")
	}
	outsider.must(http.StatusNoContent, "POST", "/events/"+pub+"/tracks", map[string]any{"deezer_id": tracks[0]})
	tr := outsider.queueRow(pub, tracks[0])["id"].(string)
	outsider.must(http.StatusOK, "POST", "/events/"+pub+"/tracks/"+tr+"/vote", map[string]any{})

	priv := host.createEvent(map[string]any{"name": "private " + runID, "visibility": "private"})
	host.must(http.StatusNoContent, "POST", "/events/"+priv+"/tracks", map[string]any{"deezer_id": tracks[1]})
	ptr := host.queueRow(priv, tracks[1])["id"].(string)

	outsider.must(http.StatusForbidden, "POST", "/events/"+priv+"/invitations", map[string]any{"user_id": outsider.id})
	host.must(http.StatusNoContent, "POST", "/events/"+priv+"/invitations", map[string]any{"user_id": invited.id})

	if !listed(invited, priv) || !listed(host, priv) {
		t.Fatal("the host and the invited guest must find the private event")
	}
	invited.must(http.StatusOK, "POST", "/events/"+priv+"/tracks/"+ptr+"/vote", map[string]any{})

	if listed(outsider, priv) {
		t.Fatal("a private event must not be listed to someone not invited")
	}
	outsider.must(http.StatusForbidden, "GET", "/events/"+priv, nil)
	outsider.must(http.StatusForbidden, "POST", "/events/"+priv+"/tracks/"+ptr+"/vote", map[string]any{})
	outsider.must(http.StatusForbidden, "POST", "/events/"+priv+"/tracks", map[string]any{"deezer_id": tracks[2]})
	if _, st := outsider.dial("/ws/events/" + priv); st != http.StatusForbidden {
		t.Fatalf("the live room of a private event must refuse an outsider, got %d", st)
	}
}

func TestVote_LicenseOpenByDefaultThenInvitedOnly(t *testing.T) {
	host := newUser(t, "lichost")
	invited := newUser(t, "licinvited")
	outsider := newUser(t, "licoutsider")

	open := host.createEvent(map[string]any{"name": "open " + runID})
	if l := host.event(open)["license"]; l != "open" {
		t.Fatalf("the default license must let everyone vote, got %v", l)
	}

	closed := host.createEvent(map[string]any{"name": "invited only " + runID, "license": "invited_only"})
	host.must(http.StatusNoContent, "POST", "/events/"+closed+"/invitations", map[string]any{"user_id": invited.id})
	host.must(http.StatusNoContent, "POST", "/events/"+closed+"/tracks", map[string]any{"deezer_id": tracks[0]})
	tr := host.queueRow(closed, tracks[0])["id"].(string)

	outsider.must(http.StatusOK, "GET", "/events/"+closed, nil)
	outsider.must(http.StatusForbidden, "POST", "/events/"+closed+"/tracks/"+tr+"/vote", map[string]any{})
	outsider.must(http.StatusForbidden, "POST", "/events/"+closed+"/tracks", map[string]any{"deezer_id": tracks[1]})
	invited.must(http.StatusOK, "POST", "/events/"+closed+"/tracks/"+tr+"/vote", map[string]any{})
	host.must(http.StatusOK, "POST", "/events/"+closed+"/tracks/"+tr+"/vote", map[string]any{})
}

func TestVote_LicenseTimeAndPlace(t *testing.T) {
	host := newUser(t, "geohost")
	voter := newUser(t, "geovoter")

	nearby := map[string]any{"latitude": 48.8700, "longitude": 2.3650}
	lyon := map[string]any{"latitude": 45.7640, "longitude": 4.8357}

	now := time.Now()
	mk := func(name string, from, to time.Time) string {
		return host.createEvent(map[string]any{
			"name": name + " " + runID, "license": "time_and_location",
			"vote_latitude": 48.8673, "vote_longitude": 2.3631, "vote_radius_meters": 1000,
			"vote_start_at": rfc3339(from), "vote_end_at": rfc3339(to),
		})
	}

	host.must(http.StatusBadRequest, "POST", "/events", map[string]any{
		"name": "incomplete", "license": "time_and_location",
	})

	live := mk("window open", now.Add(-time.Hour), now.Add(time.Hour))
	host.must(http.StatusNoContent, "POST", "/events/"+live+"/tracks", map[string]any{"deezer_id": tracks[0]})
	tr := host.queueRow(live, tracks[0])["id"].(string)
	vote := func(where map[string]any) int {
		return voter.do("POST", "/events/"+live+"/tracks/"+tr+"/vote", where).status
	}
	if st := vote(map[string]any{}); st != http.StatusForbidden {
		t.Fatalf("no position, no vote: got %d", st)
	}
	if st := vote(lyon); st != http.StatusForbidden {
		t.Fatalf("outside the area, no vote: got %d", st)
	}
	if st := vote(nearby); st != http.StatusOK {
		t.Fatalf("inside the area and the window, the vote counts: got %d", st)
	}
	if n := host.queueRow(live, tracks[0])["vote_count"]; n != float64(1) {
		t.Fatalf("exactly the accepted vote must count, got %v", n)
	}

	for name, window := range map[string][2]time.Time{
		"window over":    {now.Add(-3 * time.Hour), now.Add(-2 * time.Hour)},
		"window not yet": {now.Add(2 * time.Hour), now.Add(3 * time.Hour)},
	} {
		ev := mk(name, window[0], window[1])
		if st := voter.do("POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": tracks[1]}).status; st != http.StatusForbidden {
			t.Fatalf("%s: suggesting must be refused, got %d", name, st)
		}
	}
}

func TestVote_ConcurrentVotesAreAllCountedExactlyOnce(t *testing.T) {
	host := newUser(t, "racehost")
	const n = 12
	voters := make([]*client, n)
	for i := range voters {
		voters[i] = newUser(t, fmt.Sprintf("racer%d", i))
	}

	ev := host.createEvent(map[string]any{"name": "race " + runID})
	a, b := tracks[2], tracks[3]
	host.must(http.StatusNoContent, "POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": a})
	host.must(http.StatusNoContent, "POST", "/events/"+ev+"/tracks", map[string]any{"deezer_id": b})
	ta := host.queueRow(ev, a)["id"].(string)
	tb := host.queueRow(ev, b)["id"].(string)

	var wg sync.WaitGroup
	for i, v := range voters {
		mine, other := ta, tb
		if i%2 == 1 {
			mine, other = tb, ta
		}
		for _, tr := range []string{mine, mine, other} {
			wg.Add(1)
			go func(v *client, tr string) {
				defer wg.Done()
				if r := v.do("POST", "/events/"+ev+"/tracks/"+tr+"/vote", map[string]any{}); r.status != http.StatusOK {
					t.Errorf("concurrent vote: %s", r)
				}
			}(v, tr)
		}
	}
	wg.Wait()

	for _, tr := range []int64{a, b} {
		if got := host.queueRow(ev, tr)["vote_count"]; got != float64(n) {
			t.Fatalf("each of the %d voters counts once per track, track %d has %v", n, tr, got)
		}
	}
}
