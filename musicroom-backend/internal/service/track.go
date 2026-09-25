package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/nfauconn/music-room/musicroom-backend/internal/deezer"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

func cacheTrackByID(ctx context.Context, q *repository.Queries, dz *deezer.Client, deezerID int64) error {
	if _, err := q.GetTrack(ctx, deezerID); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	t, err := dz.GetTrack(ctx, deezerID)
	if err != nil {
		return ErrTrackNotFound
	}
	_, err = q.UpsertTrack(ctx, toUpsertParams(t))
	return err
}

type TrackService struct {
	deezer *deezer.Client
	q      *repository.Queries
}

func NewTrackService(dz *deezer.Client, q *repository.Queries) *TrackService {
	return &TrackService{deezer: dz, q: q}
}

func (s *TrackService) Search(ctx context.Context, query string, limit int) ([]deezer.Track, error) {
	tracks, err := s.deezer.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	for _, t := range tracks {
		_, _ = s.q.UpsertTrack(ctx, toUpsertParams(t))
	}
	return tracks, nil
}

func (s *TrackService) FreshPreview(ctx context.Context, deezerID int64) (string, error) {
	t, err := s.deezer.GetTrack(ctx, deezerID)
	if err != nil {
		return "", ErrTrackNotFound
	}
	_, _ = s.q.UpsertTrack(ctx, toUpsertParams(t))
	return t.PreviewURL, nil
}
