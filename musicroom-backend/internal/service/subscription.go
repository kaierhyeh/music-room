package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

var (
	ErrPremiumRequired = errors.New("premium subscription required")
	ErrUnknownTier     = errors.New("unknown subscription tier")
)

const premiumTerm = 30 * 24 * time.Hour

type SubscriptionService struct {
	pool *pgxpool.Pool
	q    *repository.Queries
}

func NewSubscriptionService(pool *pgxpool.Pool) *SubscriptionService {
	return &SubscriptionService{pool: pool, q: repository.New(pool)}
}

type Subscription struct {
	Tier      string     `json:"tier"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (s *SubscriptionService) Get(ctx context.Context, userID uuid.UUID) (Subscription, error) {
	row, err := s.q.GetSubscription(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, ErrUserNotFound
		}
		return Subscription{}, err
	}
	return subscriptionFrom(row.SubscriptionTier, row.SubscriptionExpiresAt.Time, row.SubscriptionExpiresAt.Valid), nil
}

func (s *SubscriptionService) Set(ctx context.Context, userID uuid.UUID, tier, note string) (Subscription, error) {
	var (
		t       repository.SubscriptionTier
		expires pgtype.Timestamptz
	)
	switch tier {
	case string(repository.SubscriptionTierPremium):
		t = repository.SubscriptionTierPremium
		expires = ts(time.Now().Add(premiumTerm))
	case string(repository.SubscriptionTierFree):
		t = repository.SubscriptionTierFree
	default:
		return Subscription{}, ErrUnknownTier
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Subscription{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	row, err := qtx.SetSubscription(ctx, repository.SetSubscriptionParams{
		ID:                    userID,
		SubscriptionTier:      t,
		SubscriptionExpiresAt: expires,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Subscription{}, ErrUserNotFound
		}
		return Subscription{}, err
	}
	var n *string
	if note != "" {
		n = &note
	}
	if err := qtx.RecordSubscriptionChange(ctx, repository.RecordSubscriptionChangeParams{
		UserID: userID, Tier: t, Note: n,
	}); err != nil {
		return Subscription{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Subscription{}, err
	}
	return subscriptionFrom(row.SubscriptionTier, row.SubscriptionExpiresAt.Time, row.SubscriptionExpiresAt.Valid), nil
}

type HistoryEntry struct {
	Tier      string    `json:"tier"`
	Note      *string   `json:"note,omitempty"`
	ChangedAt time.Time `json:"changed_at"`
}

func (s *SubscriptionService) History(ctx context.Context, userID uuid.UUID) ([]HistoryEntry, error) {
	rows, err := s.q.ListSubscriptionHistory(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, HistoryEntry{
			Tier:      string(r.Tier),
			Note:      r.Note,
			ChangedAt: r.ChangedAt.Time,
		})
	}
	return out, nil
}

func subscriptionFrom(tier repository.SubscriptionTier, expires time.Time, hasExpiry bool) Subscription {
	out := Subscription{Tier: string(tier)}
	if tier == repository.SubscriptionTierPremium && hasExpiry {
		if expires.Before(time.Now()) {
			return Subscription{Tier: string(repository.SubscriptionTierFree)}
		}
		e := expires
		out.ExpiresAt = &e
	}
	return out
}

func requirePremium(ctx context.Context, q *repository.Queries, userID uuid.UUID) error {
	tier, err := q.GetEffectiveTier(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if tier != repository.SubscriptionTierPremium {
		return ErrPremiumRequired
	}
	return nil
}
