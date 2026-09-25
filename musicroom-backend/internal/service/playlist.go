package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nfauconn/music-room/musicroom-backend/internal/deezer"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

const positionMinGap = 1e-6

var (
	ErrPlaylistNotFound       = errors.New("playlist not found")
	ErrPlaylistTrackNotFound  = errors.New("track not found in playlist")
	ErrVersionConflict        = errors.New("version conflict (refresh and retry)")
	ErrTrackAlreadyInPlaylist = errors.New("this track is already in the playlist")
	ErrInvalidMove            = errors.New("invalid move")
)

type RoomCloser interface {
	CloseRoom(room uuid.UUID)
}

type PlaylistBroadcaster interface {
	Broadcaster
	RoomCloser
}

type PlaylistService struct {
	pool   *pgxpool.Pool
	q      *repository.Queries
	deezer *deezer.Client
	bc     PlaylistBroadcaster
}

func NewPlaylistService(pool *pgxpool.Pool, dz *deezer.Client, bc PlaylistBroadcaster) *PlaylistService {
	return &PlaylistService{pool: pool, q: repository.New(pool), deezer: dz, bc: bc}
}

type Invite struct {
	UserID  uuid.UUID
	CanEdit bool
}

func (s *PlaylistService) Create(ctx context.Context, ownerID uuid.UUID, name, description string, vis repository.Visibility, lic repository.PlaylistLicense, invites []Invite) (repository.Playlist, string, error) {
	var desc *string
	if description != "" {
		desc = &description
	}
	for _, inv := range invites {
		if inv.CanEdit {
			if err := requirePremium(ctx, s.q, ownerID); err != nil {
				return repository.Playlist{}, "", err
			}
			break
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return repository.Playlist{}, "", err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	pl, err := qtx.CreatePlaylist(ctx, repository.CreatePlaylistParams{
		OwnerID: ownerID, Name: name, Description: desc, Visibility: vis, License: lic,
	})
	if err != nil {
		return repository.Playlist{}, "", err
	}
	for _, inv := range invites {
		if inv.UserID == ownerID {
			continue
		}
		if err := qtx.AddPlaylistCollaborator(ctx, repository.AddPlaylistCollaboratorParams{
			PlaylistID: pl.ID, UserID: inv.UserID, InvitedBy: pgtype.UUID{Bytes: ownerID, Valid: true},
			CanEdit: inv.CanEdit,
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return repository.Playlist{}, "", ErrUserNotFound
			}
			return repository.Playlist{}, "", err
		}
	}
	ownerName, err := qtx.GetUserDisplayName(ctx, ownerID)
	if err != nil {
		return repository.Playlist{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return repository.Playlist{}, "", err
	}
	return pl, ownerName, nil
}

func (s *PlaylistService) ListPublic(ctx context.Context, limit, offset int32) ([]repository.ListPublicPlaylistsRow, error) {
	return s.q.ListPublicPlaylists(ctx, repository.ListPublicPlaylistsParams{Limit: limit, Offset: offset})
}

func (s *PlaylistService) ListMine(ctx context.Context, ownerID uuid.UUID, limit, offset int32) ([]repository.ListPlaylistsOwnedByRow, error) {
	return s.q.ListPlaylistsOwnedBy(ctx, repository.ListPlaylistsOwnedByParams{OwnerID: ownerID, Limit: limit, Offset: offset})
}

func (s *PlaylistService) ListShared(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]repository.ListPlaylistsSharedWithRow, error) {
	return s.q.ListPlaylistsSharedWith(ctx, repository.ListPlaylistsSharedWithParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *PlaylistService) Get(ctx context.Context, viewerID, playlistID uuid.UUID) (repository.Playlist, string, []repository.GetPlaylistTracksRow, error) {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return repository.Playlist{}, "", nil, err
	}
	if ok, err := s.canView(ctx, pl, viewerID); err != nil {
		return repository.Playlist{}, "", nil, err
	} else if !ok {
		return repository.Playlist{}, "", nil, ErrForbidden
	}
	ownerName, err := s.q.GetUserDisplayName(ctx, pl.OwnerID)
	if err != nil {
		return repository.Playlist{}, "", nil, err
	}
	tracks, err := s.q.GetPlaylistTracks(ctx, playlistID)
	if err != nil {
		return repository.Playlist{}, "", nil, err
	}
	return pl, ownerName, tracks, nil
}

type PlaylistUpdate struct {
	Name        *string
	Description *string
	Visibility  *repository.Visibility
	License     *repository.PlaylistLicense
}

func (s *PlaylistService) Update(ctx context.Context, userID, playlistID uuid.UUID, upd PlaylistUpdate) (repository.Playlist, string, error) {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return repository.Playlist{}, "", err
	}
	if pl.OwnerID != userID {
		return repository.Playlist{}, "", ErrForbidden
	}
	params := repository.UpdatePlaylistParams{
		ID: playlistID, Name: pl.Name, Description: pl.Description,
		Visibility: pl.Visibility, License: pl.License,
	}
	if upd.Name != nil {
		params.Name = *upd.Name
	}
	if upd.Description != nil {
		if *upd.Description == "" {
			params.Description = nil
		} else {
			params.Description = upd.Description
		}
	}
	if upd.Visibility != nil {
		params.Visibility = *upd.Visibility
	}
	if upd.License != nil {
		params.License = *upd.License
	}
	updated, err := s.q.UpdatePlaylist(ctx, params)
	if err != nil {
		return repository.Playlist{}, "", err
	}
	ownerName, err := s.q.GetUserDisplayName(ctx, updated.OwnerID)
	if err != nil {
		return repository.Playlist{}, "", err
	}
	if pl.Visibility == repository.VisibilityPublic && updated.Visibility == repository.VisibilityPrivate {
		s.closeRoom(playlistID)
		return updated, ownerName, nil
	}
	s.broadcast(ctx, playlistID, updated.Version)
	return updated, ownerName, nil
}

func (s *PlaylistService) AddTrack(ctx context.Context, userID, playlistID uuid.UUID, deezerID int64, after *uuid.UUID) (int32, error) {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	if err := s.requireEdit(ctx, pl, userID); err != nil {
		return 0, err
	}
	if err := cacheTrackByID(ctx, s.q, s.deezer, deezerID); err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if _, err := qtx.GetPlaylistForUpdate(ctx, playlistID); err != nil {
		return 0, err
	}
	rows, err := qtx.ListTrackPositions(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	pos, err := placeTrack(ctx, qtx, rows, after)
	if err != nil {
		return 0, err
	}
	if _, err := qtx.AddPlaylistTrack(ctx, repository.AddPlaylistTrackParams{
		PlaylistID: playlistID, TrackDeezerID: deezerID, Position: pos,
		AddedBy: pgtype.UUID{Bytes: userID, Valid: true},
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrTrackAlreadyInPlaylist
		}
		return 0, err
	}
	ver, err := qtx.BumpPlaylistVersion(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	s.broadcast(ctx, playlistID, ver)
	return ver, nil
}

func (s *PlaylistService) MoveTrack(ctx context.Context, userID, playlistID, trackID uuid.UUID, after *uuid.UUID, expectedVersion int32) (int32, error) {
	if after != nil && *after == trackID {
		return 0, fmt.Errorf("%w: a track cannot be placed after itself", ErrInvalidMove)
	}
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	if err := s.requireEdit(ctx, pl, userID); err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	locked, err := qtx.GetPlaylistForUpdate(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrPlaylistNotFound
		}
		return 0, err
	}
	if locked.Version != expectedVersion {
		return 0, ErrVersionConflict
	}

	all, err := qtx.ListTrackPositions(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	others := make([]repository.ListTrackPositionsRow, 0, len(all))
	found := false
	for _, r := range all {
		if r.ID == trackID {
			found = true
			continue
		}
		others = append(others, r)
	}
	if !found {
		return 0, ErrPlaylistTrackNotFound
	}

	pos, err := placeTrack(ctx, qtx, others, after)
	if err != nil {
		return 0, err
	}
	if err := qtx.SetTrackPosition(ctx, repository.SetTrackPositionParams{ID: trackID, Position: pos}); err != nil {
		return 0, err
	}
	ver, err := qtx.BumpPlaylistVersion(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	s.broadcast(ctx, playlistID, ver)
	return ver, nil
}

func (s *PlaylistService) RemoveTrack(ctx context.Context, userID, playlistID, trackID uuid.UUID) (int32, error) {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	if err := s.requireEdit(ctx, pl, userID); err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)
	if _, err := qtx.GetPlaylistForUpdate(ctx, playlistID); err != nil {
		return 0, err
	}
	n, err := qtx.RemovePlaylistTrack(ctx, repository.RemovePlaylistTrackParams{PlaylistID: playlistID, ID: trackID})
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, ErrPlaylistTrackNotFound
	}
	ver, err := qtx.BumpPlaylistVersion(ctx, playlistID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	s.broadcast(ctx, playlistID, ver)
	return ver, nil
}

func (s *PlaylistService) AddCollaborator(ctx context.Context, ownerID, playlistID, userID uuid.UUID, canEdit bool) error {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return err
	}
	if pl.OwnerID != ownerID {
		return ErrForbidden
	}
	if canEdit {
		if err := requirePremium(ctx, s.q, ownerID); err != nil {
			return err
		}
	}
	if err := s.q.AddPlaylistCollaborator(ctx, repository.AddPlaylistCollaboratorParams{
		PlaylistID: playlistID, UserID: userID, InvitedBy: pgtype.UUID{Bytes: ownerID, Valid: true},
		CanEdit: canEdit,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *PlaylistService) RemoveCollaborator(ctx context.Context, ownerID, playlistID, userID uuid.UUID) error {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return err
	}
	if pl.OwnerID != ownerID {
		return ErrForbidden
	}
	n, err := s.q.RemovePlaylistCollaborator(ctx, repository.RemovePlaylistCollaboratorParams{
		PlaylistID: playlistID, UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	if pl.Visibility == repository.VisibilityPrivate {
		s.closeRoom(playlistID)
	}
	return nil
}

func (s *PlaylistService) ListCollaborators(ctx context.Context, viewerID, playlistID uuid.UUID) ([]repository.ListPlaylistCollaboratorsRow, error) {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if ok, err := s.canView(ctx, pl, viewerID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrForbidden
	}
	return s.q.ListPlaylistCollaborators(ctx, playlistID)
}

func (s *PlaylistService) CanAccessPlaylistRoom(ctx context.Context, playlistID, userID uuid.UUID) (bool, error) {
	pl, err := s.getPlaylist(ctx, playlistID)
	if err != nil {
		return false, err
	}
	return s.canView(ctx, pl, userID)
}

func (s *PlaylistService) getPlaylist(ctx context.Context, playlistID uuid.UUID) (repository.Playlist, error) {
	pl, err := s.q.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Playlist{}, ErrPlaylistNotFound
		}
		return repository.Playlist{}, err
	}
	return pl, nil
}

func (s *PlaylistService) isOwnerOrCollaborator(ctx context.Context, pl repository.Playlist, userID uuid.UUID) (bool, error) {
	if pl.OwnerID == userID {
		return true, nil
	}
	return s.q.IsPlaylistCollaborator(ctx, repository.IsPlaylistCollaboratorParams{PlaylistID: pl.ID, UserID: userID})
}

func (s *PlaylistService) canView(ctx context.Context, pl repository.Playlist, userID uuid.UUID) (bool, error) {
	if pl.Visibility == repository.VisibilityPublic {
		return true, nil
	}
	return s.isOwnerOrCollaborator(ctx, pl, userID)
}

func (s *PlaylistService) requireEdit(ctx context.Context, pl repository.Playlist, userID uuid.UUID) error {
	ok, err := s.canView(ctx, pl, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	if pl.OwnerID == userID {
		return nil
	}
	canEdit, err := s.q.GetPlaylistCollaboratorRole(ctx, repository.GetPlaylistCollaboratorRoleParams{PlaylistID: pl.ID, UserID: userID})
	switch {
	case err == nil:
		if !canEdit {
			return ErrForbidden
		}
	case errors.Is(err, pgx.ErrNoRows):
		if pl.License == repository.PlaylistLicenseInvitedOnly {
			return ErrForbidden
		}
	default:
		return err
	}
	return requirePremium(ctx, s.q, userID)
}

func (s *PlaylistService) closeRoom(playlistID uuid.UUID) {
	if s.bc == nil {
		return
	}
	s.bc.CloseRoom(playlistID)
}

func (s *PlaylistService) broadcast(ctx context.Context, playlistID uuid.UUID, version int32) {
	if s.bc == nil {
		return
	}
	tracks, err := s.q.GetPlaylistTracks(ctx, playlistID)
	if err != nil {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":        "playlist:updated",
		"playlist_id": playlistID,
		"version":     version,
		"tracks":      tracks,
	})
	if err != nil {
		return
	}
	s.bc.Broadcast(playlistID, msg)
}

func placeTrack(ctx context.Context, qtx *repository.Queries, others []repository.ListTrackPositionsRow, after *uuid.UUID) (float64, error) {
	idx, err := afterIndex(others, after)
	if err != nil {
		return 0, err
	}
	pos, needRenorm := computePosition(positionsOf(others), idx)
	if !needRenorm {
		return pos, nil
	}
	for i, r := range others {
		if err := qtx.SetTrackPosition(ctx, repository.SetTrackPositionParams{ID: r.ID, Position: float64(i + 1)}); err != nil {
			return 0, err
		}
	}
	switch {
	case idx < 0:
		return 0.5, nil
	case idx == len(others)-1:
		return float64(len(others) + 1), nil
	default:
		return float64(idx+1) + 0.5, nil
	}
}

func afterIndex(others []repository.ListTrackPositionsRow, after *uuid.UUID) (int, error) {
	if after == nil {
		return -1, nil
	}
	for i, r := range others {
		if r.ID == *after {
			return i, nil
		}
	}
	return 0, ErrPlaylistTrackNotFound
}

func positionsOf(rows []repository.ListTrackPositionsRow) []float64 {
	ps := make([]float64, len(rows))
	for i, r := range rows {
		ps[i] = r.Position
	}
	return ps
}

func computePosition(positions []float64, afterIdx int) (float64, bool) {
	if afterIdx < 0 {
		if len(positions) == 0 {
			return 1.0, false
		}
		return positions[0] - 1.0, false
	}
	lower := positions[afterIdx]
	if afterIdx == len(positions)-1 {
		return lower + 1.0, false
	}
	upper := positions[afterIdx+1]
	return (lower + upper) / 2, (upper - lower) < 2*positionMinGap
}
