//go:build e2e

package e2e

import (
	"net/http"
	"slices"
	"sync"
	"testing"
)

func TestPlaylist_RealTimeMultiUserEditing(t *testing.T) {
	owner := newUser(t, "plrt")
	friend := newUser(t, "plrtfriend")
	friend.premium()

	pl := owner.createPlaylist(map[string]any{"name": "radio " + runID})
	ownerRoom, st := owner.dial("/ws/playlists/" + pl)
	if st != http.StatusSwitchingProtocols {
		t.Fatalf("the owner must join the live room, got %d", st)
	}
	friendRoom, _ := friend.dial("/ws/playlists/" + pl)

	friend.must(http.StatusCreated, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[0]})
	msg := ownerRoom.next("playlist:updated")
	if got, _ := msg["tracks"].([]any); len(got) != 1 {
		t.Fatalf("the owner must receive the friend's addition live, got %v", msg)
	}

	owner.must(http.StatusCreated, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[1]})
	friendRoom.next("playlist:updated")
	msg = friendRoom.next("playlist:updated")
	if got, _ := msg["tracks"].([]any); len(got) != 2 {
		t.Fatalf("the friend must receive the owner's addition live, got %v", msg)
	}

	friend.must(http.StatusConflict, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": tracks[0]})

	rows, v := owner.playlist(pl)
	friend.must(http.StatusOK, "DELETE", "/playlists/"+pl+"/tracks/"+rows[0]["id"].(string), nil)
	if rows, v2 := owner.playlist(pl); len(rows) != 1 || v2 <= v {
		t.Fatalf("a removal must shrink the playlist and bump its version, got %d rows v%d→v%d", len(rows), v, v2)
	}
}

func TestPlaylist_PublicByDefaultPrivateForInvitedOnly(t *testing.T) {
	owner := newUser(t, "plvis")
	invited := newUser(t, "plvisinv")
	stranger := newUser(t, "plvisstr")

	pub := owner.createPlaylist(map[string]any{"name": "default " + runID})
	m := owner.must(http.StatusOK, "GET", "/playlists/"+pub, nil).obj(t)
	if m["visibility"] != "public" {
		t.Fatalf("a playlist must be public by default, got %v", m["visibility"])
	}
	if !contains(stranger.must(http.StatusOK, "GET", "/playlists", nil).list(t), pub) {
		t.Fatal("a public playlist must be listed to everyone")
	}
	stranger.must(http.StatusOK, "GET", "/playlists/"+pub, nil)

	priv := owner.createPlaylist(map[string]any{"name": "private " + runID, "visibility": "private"})
	owner.must(http.StatusNoContent, "POST", "/playlists/"+priv+"/collaborators", map[string]any{"user_id": invited.id, "can_edit": false})

	if contains(stranger.must(http.StatusOK, "GET", "/playlists", nil).list(t), priv) {
		t.Fatal("a private playlist must not be listed")
	}
	stranger.must(http.StatusForbidden, "GET", "/playlists/"+priv, nil)
	if _, st := stranger.dial("/ws/playlists/" + priv); st != http.StatusForbidden {
		t.Fatalf("the live room of a private playlist must refuse a stranger, got %d", st)
	}

	invited.must(http.StatusOK, "GET", "/playlists/"+priv, nil)
	if !contains(invited.must(http.StatusOK, "GET", "/playlists/shared", nil).list(t), priv) {
		t.Fatal("the invited user must find the private playlist among those shared with them")
	}
	if _, st := invited.dial("/ws/playlists/" + priv); st != http.StatusSwitchingProtocols {
		t.Fatalf("the invited user must join the live room, got %d", st)
	}
}

func TestPlaylist_LicenseOpenByDefaultThenInvitedOnly(t *testing.T) {
	owner := newUser(t, "pllic")
	owner.premium()
	invited := newUser(t, "pllicinv")
	invited.premium()
	stranger := newUser(t, "pllicstr")
	stranger.premium()

	open := owner.createPlaylist(map[string]any{"name": "open " + runID})
	if l := owner.must(http.StatusOK, "GET", "/playlists/"+open, nil).obj(t)["license"]; l != "open" {
		t.Fatalf("the default license must let everyone edit, got %v", l)
	}
	stranger.must(http.StatusCreated, "POST", "/playlists/"+open+"/tracks", map[string]any{"deezer_id": tracks[0]})

	closed := owner.createPlaylist(map[string]any{"name": "invited only " + runID, "license": "invited_only"})
	owner.must(http.StatusNoContent, "POST", "/playlists/"+closed+"/collaborators", map[string]any{"user_id": invited.id, "can_edit": true})

	stranger.must(http.StatusOK, "GET", "/playlists/"+closed, nil)
	stranger.must(http.StatusForbidden, "POST", "/playlists/"+closed+"/tracks", map[string]any{"deezer_id": tracks[1]})
	invited.must(http.StatusCreated, "POST", "/playlists/"+closed+"/tracks", map[string]any{"deezer_id": tracks[1]})
	owner.must(http.StatusCreated, "POST", "/playlists/"+closed+"/tracks", map[string]any{"deezer_id": tracks[2]})

	rows, v := owner.playlist(closed)
	stranger.must(http.StatusForbidden, "PATCH", "/playlists/"+closed+"/tracks/"+rows[0]["id"].(string), map[string]any{"version": v})
	stranger.must(http.StatusForbidden, "DELETE", "/playlists/"+closed+"/tracks/"+rows[0]["id"].(string), nil)

	invited.must(http.StatusForbidden, "PATCH", "/playlists/"+closed, map[string]any{"license": "open"})
}

func TestPlaylist_ConcurrentMovesNeverCorruptTheOrder(t *testing.T) {
	owner := newUser(t, "plrace")
	owner.premium()
	editors := []*client{owner, newUser(t, "plrace1"), newUser(t, "plrace2"), newUser(t, "plrace3")}
	for _, e := range editors[1:] {
		e.premium()
	}

	pl := owner.createPlaylist(map[string]any{"name": "race " + runID})
	for _, id := range tracks[:6] {
		owner.must(http.StatusCreated, "POST", "/playlists/"+pl+"/tracks", map[string]any{"deezer_id": id})
	}
	rows, v := owner.playlist(pl)
	before := deezerIDs(rows)

	moves := []struct{ track, after string }{
		{rows[5]["id"].(string), ""},
		{rows[5]["id"].(string), rows[2]["id"].(string)},
		{rows[0]["id"].(string), rows[4]["id"].(string)},
		{rows[3]["id"].(string), rows[1]["id"].(string)},
	}
	statuses := make([]int, len(moves))
	var wg sync.WaitGroup
	for i, mv := range moves {
		wg.Add(1)
		go func(i int, c *client) {
			defer wg.Done()
			body := map[string]any{"version": v}
			if mv.after != "" {
				body["after_track_id"] = mv.after
			}
			statuses[i] = c.do("PATCH", "/playlists/"+pl+"/tracks/"+mv.track, body).status
		}(i, editors[i])
	}
	wg.Wait()

	won := 0
	for _, st := range statuses {
		switch st {
		case http.StatusOK:
			won++
		case http.StatusConflict:
		default:
			t.Fatalf("a concurrent move must succeed or report a conflict, got %v", statuses)
		}
	}
	if won != 1 {
		t.Fatalf("against one version exactly one move may land, got %v", statuses)
	}

	after, v2 := owner.playlist(pl)
	if v2 != v+1 {
		t.Fatalf("one landed move is one version bump: v%d → v%d", v, v2)
	}
	got := deezerIDs(after)
	slices.Sort(got)
	slices.Sort(before)
	if !slices.Equal(got, before) {
		t.Fatalf("moves must neither lose nor duplicate a track: %v vs %v", got, before)
	}

	editors[1].must(http.StatusOK, "PATCH", "/playlists/"+pl+"/tracks/"+moves[1].track,
		map[string]any{"version": v2, "after_track_id": moves[1].after})
}
