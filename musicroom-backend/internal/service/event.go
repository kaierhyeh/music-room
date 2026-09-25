package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nfauconn/music-room/musicroom-backend/internal/deezer"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

var (
	ErrEventNotFound      = errors.New("event not found")
	ErrEventTrackNotFound = errors.New("track not found in this event")
	ErrTrackNotFound      = errors.New("Deezer track not found")
	ErrForbidden          = errors.New("access denied")
	ErrInvalidEvent       = errors.New("invalid event parameters")
	ErrVoteNotAllowed     = errors.New("vote not allowed for this event")
	ErrVoteWindowClosed   = errors.New("voting is not open now")
	ErrLocationRequired   = errors.New("GPS location required to vote")
	ErrOutsideVoteArea    = errors.New("outside the authorized voting area")
)

type Broadcaster interface {
	Broadcast(room uuid.UUID, msg []byte)
}

type EventService struct {
	pool   *pgxpool.Pool
	q      *repository.Queries
	deezer *deezer.Client
	bc     Broadcaster
}

func NewEventService(pool *pgxpool.Pool, dz *deezer.Client, bc Broadcaster) *EventService {
	return &EventService{pool: pool, q: repository.New(pool), deezer: dz, bc: bc}
}

type CreateEventInput struct {
	OwnerID          uuid.UUID
	Name             string
	Description      string
	Visibility       repository.Visibility
	License          repository.EventLicense
	VoteStartAt      *time.Time
	VoteEndAt        *time.Time
	VoteLatitude     *float64
	VoteLongitude    *float64
	VoteRadiusMeters *int
}

func (s *EventService) CreateEvent(ctx context.Context, in CreateEventInput) (repository.Event, error) {
	if in.License == repository.EventLicenseTimeAndLocation {
		if in.VoteStartAt == nil || in.VoteEndAt == nil || in.VoteLatitude == nil || in.VoteLongitude == nil || in.VoteRadiusMeters == nil {
			return repository.Event{}, fmt.Errorf("%w: time_and_location license requires start/end/lat/lng/radius", ErrInvalidEvent)
		}
		if !in.VoteEndAt.After(*in.VoteStartAt) {
			return repository.Event{}, fmt.Errorf("%w: vote_end_at must be after vote_start_at", ErrInvalidEvent)
		}
		if *in.VoteRadiusMeters <= 0 {
			return repository.Event{}, fmt.Errorf("%w: vote_radius_meters must be > 0", ErrInvalidEvent)
		}
	}
	var desc *string
	if in.Description != "" {
		desc = &in.Description
	}
	return s.q.CreateEvent(ctx, repository.CreateEventParams{
		OwnerID:          in.OwnerID,
		Name:             in.Name,
		Description:      desc,
		Visibility:       in.Visibility,
		License:          in.License,
		VoteStartAt:      tsPtr(in.VoteStartAt),
		VoteEndAt:        tsPtr(in.VoteEndAt),
		VoteLatitude:     in.VoteLatitude,
		VoteLongitude:    in.VoteLongitude,
		VoteRadiusMeters: i32FromIntPtr(in.VoteRadiusMeters),
	})
}

func (s *EventService) ListVisible(ctx context.Context, viewerID uuid.UUID, limit, offset int32) ([]repository.ListVisibleEventsRow, error) {
	return s.q.ListVisibleEvents(ctx, repository.ListVisibleEventsParams{ViewerID: viewerID, Limit: limit, Offset: offset})
}

func (s *EventService) GetEvent(ctx context.Context, viewerID, eventID uuid.UUID) (repository.Event, string, []repository.GetEventQueueRow, error) {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return repository.Event{}, "", nil, err
	}
	see, err := s.canSee(ctx, ev, viewerID)
	if err != nil {
		return repository.Event{}, "", nil, err
	}
	if !see {
		return repository.Event{}, "", nil, ErrForbidden
	}
	ownerName, err := s.q.GetUserDisplayName(ctx, ev.OwnerID)
	if err != nil {
		return repository.Event{}, "", nil, err
	}
	queue, err := s.q.GetEventQueue(ctx, repository.GetEventQueueParams{
		EventID:  eventID,
		ViewerID: viewerID,
	})
	if err != nil {
		return repository.Event{}, "", nil, err
	}
	return ev, ownerName, queue, nil
}

func (s *EventService) SuggestTrack(ctx context.Context, eventID, userID uuid.UUID, deezerID int64) error {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return err
	}
	if err := s.authorizeSuggest(ctx, ev, userID); err != nil {
		return err
	}
	if err := s.ensureTrackCached(ctx, deezerID); err != nil {
		return err
	}
	if _, err := s.q.SuggestTrack(ctx, repository.SuggestTrackParams{
		EventID:       eventID,
		TrackDeezerID: deezerID,
		SuggestedBy:   pgtype.UUID{Bytes: userID, Valid: true},
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	s.broadcastQueue(ctx, eventID)
	return nil
}

func (s *EventService) CastVote(ctx context.Context, eventID, userID, eventTrackID uuid.UUID, lat, lng *float64) (bool, error) {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return false, err
	}
	if err := s.authorizeVote(ctx, ev, userID, lat, lng); err != nil {
		return false, err
	}
	if err := s.ensureTrackInEvent(ctx, eventID, eventTrackID); err != nil {
		return false, err
	}
	rows, err := s.q.CastVote(ctx, repository.CastVoteParams{EventTrackID: eventTrackID, UserID: userID})
	if err != nil {
		return false, err
	}
	if rows > 0 {
		s.broadcastQueue(ctx, eventID)
	}
	return rows > 0, nil
}

func (s *EventService) RemoveVote(ctx context.Context, eventID, userID, eventTrackID uuid.UUID) (bool, error) {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return false, err
	}
	if see, err := s.canSee(ctx, ev, userID); err != nil {
		return false, err
	} else if !see {
		return false, ErrForbidden
	}
	if err := s.requireVoteWindow(ev); err != nil {
		return false, err
	}
	if err := s.ensureTrackInEvent(ctx, eventID, eventTrackID); err != nil {
		return false, err
	}
	rows, err := s.q.RemoveVote(ctx, repository.RemoveVoteParams{EventTrackID: eventTrackID, UserID: userID})
	if err != nil {
		return false, err
	}
	if rows > 0 {
		s.broadcastQueue(ctx, eventID)
	}
	return rows > 0, nil
}

func (s *EventService) GetQueue(ctx context.Context, viewerID, eventID uuid.UUID) ([]repository.GetEventQueueRow, error) {
	return s.q.GetEventQueue(ctx, repository.GetEventQueueParams{
		EventID:  eventID,
		ViewerID: viewerID,
	})
}

func (s *EventService) InviteToEvent(ctx context.Context, ownerID, eventID, inviteeID uuid.UUID) error {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return err
	}
	if ev.OwnerID != ownerID {
		return ErrForbidden
	}
	if err := s.q.CreateEventInvitation(ctx, repository.CreateEventInvitationParams{
		EventID:   eventID,
		UserID:    inviteeID,
		InvitedBy: pgtype.UUID{Bytes: ownerID, Valid: true},
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *EventService) ListInvitations(ctx context.Context, ownerID, eventID uuid.UUID) ([]repository.ListEventInvitationsRow, error) {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if ev.OwnerID != ownerID {
		return nil, ErrForbidden
	}
	return s.q.ListEventInvitations(ctx, eventID)
}

func (s *EventService) CanAccessEventRoom(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	ev, err := s.getEvent(ctx, eventID)
	if err != nil {
		return false, err
	}
	return s.canSee(ctx, ev, userID)
}

func (s *EventService) getEvent(ctx context.Context, eventID uuid.UUID) (repository.Event, error) {
	ev, err := s.q.GetEvent(ctx, eventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Event{}, ErrEventNotFound
		}
		return repository.Event{}, err
	}
	return ev, nil
}

func (s *EventService) isOwnerOrInvited(ctx context.Context, ev repository.Event, userID uuid.UUID) (bool, error) {
	if ev.OwnerID == userID {
		return true, nil
	}
	return s.q.IsEventInvited(ctx, repository.IsEventInvitedParams{EventID: ev.ID, UserID: userID})
}

func (s *EventService) canSee(ctx context.Context, ev repository.Event, userID uuid.UUID) (bool, error) {
	if ev.Visibility == repository.VisibilityPublic {
		return true, nil
	}
	return s.isOwnerOrInvited(ctx, ev, userID)
}

func (s *EventService) authorizeSuggest(ctx context.Context, ev repository.Event, userID uuid.UUID) error {
	see, err := s.canSee(ctx, ev, userID)
	if err != nil {
		return err
	}
	if !see {
		return ErrForbidden
	}
	if ev.License == repository.EventLicenseInvitedOnly {
		ok, err := s.isOwnerOrInvited(ctx, ev, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrVoteNotAllowed
		}
	}
	return s.requireVoteWindow(ev)
}

func (s *EventService) requireVoteWindow(ev repository.Event) error {
	if ev.License != repository.EventLicenseTimeAndLocation {
		return nil
	}
	now := time.Now()
	if !ev.VoteStartAt.Valid || !ev.VoteEndAt.Valid ||
		now.Before(ev.VoteStartAt.Time) || now.After(ev.VoteEndAt.Time) {
		return ErrVoteWindowClosed
	}
	return nil
}

func (s *EventService) authorizeVote(ctx context.Context, ev repository.Event, userID uuid.UUID, lat, lng *float64) error {
	see, err := s.canSee(ctx, ev, userID)
	if err != nil {
		return err
	}
	if !see {
		return ErrForbidden
	}
	switch ev.License {
	case repository.EventLicenseOpen:
		return nil
	case repository.EventLicenseInvitedOnly:
		ok, err := s.isOwnerOrInvited(ctx, ev, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrVoteNotAllowed
		}
		return nil
	case repository.EventLicenseTimeAndLocation:
		now := time.Now()
		if !ev.VoteStartAt.Valid || !ev.VoteEndAt.Valid || now.Before(ev.VoteStartAt.Time) || now.After(ev.VoteEndAt.Time) {
			return ErrVoteWindowClosed
		}
		if lat == nil || lng == nil {
			return ErrLocationRequired
		}
		if ev.VoteLatitude == nil || ev.VoteLongitude == nil || ev.VoteRadiusMeters == nil {
			return ErrVoteNotAllowed
		}
		if haversineMeters(*lat, *lng, *ev.VoteLatitude, *ev.VoteLongitude) > float64(*ev.VoteRadiusMeters) {
			return ErrOutsideVoteArea
		}
		return nil
	}
	return ErrVoteNotAllowed
}

func (s *EventService) ensureTrackInEvent(ctx context.Context, eventID, eventTrackID uuid.UUID) error {
	et, err := s.q.GetEventTrack(ctx, eventTrackID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventTrackNotFound
		}
		return err
	}
	if et.EventID != eventID {
		return ErrEventTrackNotFound
	}
	return nil
}

func (s *EventService) ensureTrackCached(ctx context.Context, deezerID int64) error {
	return cacheTrackByID(ctx, s.q, s.deezer, deezerID)
}

func (s *EventService) broadcastQueue(ctx context.Context, eventID uuid.UUID) {
	if s.bc == nil {
		return
	}
	queue, err := s.q.GetEventQueue(ctx, repository.GetEventQueueParams{
		EventID:  eventID,
		ViewerID: uuid.Nil,
	})
	if err != nil {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"type":     "queue:updated",
		"event_id": eventID,
		"queue":    queue,
	})
	if err != nil {
		return
	}
	s.bc.Broadcast(eventID, msg)
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	const rad = math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func toUpsertParams(t deezer.Track) repository.UpsertTrackParams {
	return repository.UpsertTrackParams{
		DeezerID:    t.DeezerID,
		Title:       t.Title,
		ArtistName:  t.ArtistName,
		AlbumTitle:  strPtr(t.AlbumTitle),
		DurationSec: i32Ptr(t.DurationSec),
		PreviewUrl:  strPtr(t.PreviewURL),
		CoverUrl:    strPtr(t.CoverURL),
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func i32Ptr(i int) *int32 { v := int32(i); return &v }

func i32FromIntPtr(i *int) *int32 {
	if i == nil {
		return nil
	}
	v := int32(*i)
	return &v
}

func tsPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
