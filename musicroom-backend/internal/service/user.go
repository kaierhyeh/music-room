package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidProfile = errors.New("invalid profile data")
)

type Relationship int

const (
	RelSelf Relationship = iota
	RelFriend
	RelOther
)

type UserService struct {
	q *repository.Queries
}

func NewUserService(q *repository.Queries) *UserService { return &UserService{q: q} }

func canSee(level string, rel Relationship) bool {
	if rel == RelSelf {
		return true
	}
	switch level {
	case "public":
		return true
	case "friends":
		return rel == RelFriend
	default:
		return false
	}
}

func filterProfileData(data map[string]json.RawMessage, vis map[string]string, rel Relationship) map[string]json.RawMessage {
	if rel == RelSelf {
		return data
	}
	out := make(map[string]json.RawMessage, len(data))
	for k, v := range data {
		if canSee(vis[k], rel) {
			out[k] = v
		}
	}
	return out
}

func (s *UserService) GetProfile(ctx context.Context, viewerID, targetID uuid.UUID) (map[string]any, error) {
	u, err := s.q.GetUserByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	rel := RelOther
	switch {
	case viewerID == targetID:
		rel = RelSelf
	default:
		ok, err := s.q.AreFriends(ctx, repository.AreFriendsParams{UserA: viewerID, UserB: targetID})
		if err != nil {
			return nil, err
		}
		if ok {
			rel = RelFriend
		}
	}

	out := map[string]any{
		"id":           u.ID,
		"display_name": u.DisplayName,
	}

	if rel == RelSelf {
		linked, err := s.q.HasOAuthProvider(ctx, repository.HasOAuthProviderParams{
			UserID:   targetID,
			Provider: repository.OauthProviderGoogle,
		})
		if err != nil {
			return nil, err
		}
		out["google_linked"] = linked
		out["email"] = u.Email
		out["email_verified"] = u.EmailVerified
		tier, err := s.q.GetEffectiveTier(ctx, targetID)
		if err != nil {
			return nil, err
		}
		out["subscription_tier"] = tier
		out["profile_data"] = json.RawMessage(u.ProfileData)
		out["profile_visibility"] = json.RawMessage(u.ProfileVisibility)
		out["music_preferences"] = json.RawMessage(u.MusicPreferences)
		return out, nil
	}

	var vis map[string]string
	_ = json.Unmarshal(u.ProfileVisibility, &vis)
	var data map[string]json.RawMessage
	_ = json.Unmarshal(u.ProfileData, &data)

	out["profile_data"] = filterProfileData(data, vis, rel)
	if canSee(vis["music_preferences"], rel) {
		out["music_preferences"] = json.RawMessage(u.MusicPreferences)
	}
	return out, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, data, vis, prefs []byte) (map[string]any, error) {
	cur, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if data == nil {
		data = cur.ProfileData
	} else if !isJSONObject(data) {
		return nil, fmt.Errorf("%w: profile_data must be a JSON object", ErrInvalidProfile)
	}
	if prefs == nil {
		prefs = cur.MusicPreferences
	} else if !isJSONObject(prefs) {
		return nil, fmt.Errorf("%w: music_preferences must be a JSON object", ErrInvalidProfile)
	}
	if vis == nil {
		vis = cur.ProfileVisibility
	} else if err := validateVisibility(vis); err != nil {
		return nil, err
	}

	if _, err := s.q.UpdateProfile(ctx, repository.UpdateProfileParams{
		ID:                userID,
		ProfileData:       data,
		ProfileVisibility: vis,
		MusicPreferences:  prefs,
	}); err != nil {
		return nil, err
	}
	return s.GetProfile(ctx, userID, userID)
}

func isJSONObject(b []byte) bool {
	var m map[string]json.RawMessage
	return json.Unmarshal(b, &m) == nil
}

func validateVisibility(b []byte) error {
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("%w: profile_visibility must be a field->level object", ErrInvalidProfile)
	}
	for field, level := range m {
		switch level {
		case "public", "friends", "private":
		default:
			return fmt.Errorf("%w: invalid level %q for %q (public|friends|private)", ErrInvalidProfile, level, field)
		}
	}
	return nil
}
