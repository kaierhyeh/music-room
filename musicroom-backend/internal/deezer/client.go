package deezer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Track struct {
	DeezerID    int64  `json:"deezer_id"`
	Title       string `json:"title"`
	ArtistName  string `json:"artist_name"`
	AlbumTitle  string `json:"album_title"`
	DurationSec int    `json:"duration_sec"`
	PreviewURL  string `json:"preview_url"`
	CoverURL    string `json:"cover_url"`
}

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client {
	if base == "" {
		base = "https://api.deezer.com"
	}
	return &Client{base: base, http: &http.Client{Timeout: 8 * time.Second}}
}

type apiTrack struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Preview  string `json:"preview"`
	Artist   struct {
		Name string `json:"name"`
	} `json:"artist"`
	Album struct {
		Title       string `json:"title"`
		CoverMedium string `json:"cover_medium"`
	} `json:"album"`
}

func (t apiTrack) toTrack() Track {
	return Track{
		DeezerID:    t.ID,
		Title:       t.Title,
		ArtistName:  t.Artist.Name,
		AlbumTitle:  t.Album.Title,
		DurationSec: t.Duration,
		PreviewURL:  t.Preview,
		CoverURL:    t.Album.CoverMedium,
	}
}

func (c *Client) Search(ctx context.Context, q string, limit int) ([]Track, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	u := fmt.Sprintf("%s/search?q=%s&limit=%d", c.base, url.QueryEscape(q), limit)
	var body struct {
		Data []apiTrack `json:"data"`
	}
	if err := c.get(ctx, u, &body); err != nil {
		return nil, err
	}
	out := make([]Track, 0, len(body.Data))
	for _, t := range body.Data {
		out = append(out, t.toTrack())
	}
	return out, nil
}

func (c *Client) GetTrack(ctx context.Context, id int64) (Track, error) {
	var t apiTrack
	if err := c.get(ctx, fmt.Sprintf("%s/track/%d", c.base, id), &t); err != nil {
		return Track{}, err
	}
	if t.ID == 0 {
		return Track{}, fmt.Errorf("deezer: track %d not found", id)
	}
	return t.toTrack(), nil
}

func (c *Client) get(ctx context.Context, u string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deezer: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
