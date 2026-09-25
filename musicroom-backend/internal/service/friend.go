package service

import (
	"bytes"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

var (
	ErrCannotFriendSelf = errors.New("cannot add yourself as a friend")
	ErrAlreadyFriends   = errors.New("already friends")
	ErrRequestExists    = errors.New("request already pending")
	ErrRequestNotFound  = errors.New("request not found")
	ErrNotAddressee     = errors.New("unauthorized action")
)

type FriendService struct {
	pool *pgxpool.Pool
	q    *repository.Queries
}

func NewFriendService(pool *pgxpool.Pool) *FriendService {
	return &FriendService{pool: pool, q: repository.New(pool)}
}

func (s *FriendService) SendRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) (string, error) {
	if requesterID == addresseeID {
		return "", ErrCannotFriendSelf
	}
	friends, err := s.q.AreFriends(ctx, repository.AreFriendsParams{UserA: requesterID, UserB: addresseeID})
	if err != nil {
		return "", err
	}
	if friends {
		return "", ErrAlreadyFriends
	}

	pending, err := s.q.GetPendingRequestBetween(ctx, repository.GetPendingRequestBetweenParams{U1: requesterID, U2: addresseeID})
	switch {
	case err == nil:
		if pending.RequesterID == addresseeID {
			if err := s.acceptInternal(ctx, pending); err != nil {
				return "", err
			}
			return "accepted", nil
		}
		return "", ErrRequestExists
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return "", err
	}

	if _, err := s.q.CreateFriendRequest(ctx, repository.CreateFriendRequestParams{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				return "", ErrUserNotFound
			case "23505":
				return "", ErrRequestExists
			}
		}
		return "", err
	}
	return "pending", nil
}

func (s *FriendService) Accept(ctx context.Context, userID, requestID uuid.UUID) error {
	fr, err := s.getPendingFor(ctx, userID, requestID)
	if err != nil {
		return err
	}
	return s.acceptInternal(ctx, fr)
}

func (s *FriendService) Reject(ctx context.Context, userID, requestID uuid.UUID) error {
	fr, err := s.getPendingFor(ctx, userID, requestID)
	if err != nil {
		return err
	}
	return s.q.SetFriendRequestStatus(ctx, repository.SetFriendRequestStatusParams{
		ID:     fr.ID,
		Status: repository.FriendRequestStatusRejected,
	})
}

func (s *FriendService) getPendingFor(ctx context.Context, userID, requestID uuid.UUID) (repository.FriendRequest, error) {
	fr, err := s.q.GetFriendRequest(ctx, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.FriendRequest{}, ErrRequestNotFound
		}
		return repository.FriendRequest{}, err
	}
	if fr.AddresseeID != userID {
		return repository.FriendRequest{}, ErrNotAddressee
	}
	if fr.Status != repository.FriendRequestStatusPending {
		return repository.FriendRequest{}, ErrRequestNotFound
	}
	return fr, nil
}

func (s *FriendService) acceptInternal(ctx context.Context, fr repository.FriendRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if err := qtx.SetFriendRequestStatus(ctx, repository.SetFriendRequestStatusParams{
		ID:     fr.ID,
		Status: repository.FriendRequestStatusAccepted,
	}); err != nil {
		return err
	}
	a, b := canonical(fr.RequesterID, fr.AddresseeID)
	if err := qtx.CreateFriendship(ctx, repository.CreateFriendshipParams{UserIDA: a, UserIDB: b}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *FriendService) ListFriends(ctx context.Context, userID uuid.UUID) ([]repository.ListFriendsRow, error) {
	return s.q.ListFriends(ctx, userID)
}

func (s *FriendService) ListIncoming(ctx context.Context, userID uuid.UUID) ([]repository.ListIncomingRequestsRow, error) {
	return s.q.ListIncomingRequests(ctx, userID)
}

func canonical(x, y uuid.UUID) (uuid.UUID, uuid.UUID) {
	if bytes.Compare(x[:], y[:]) < 0 {
		return x, y
	}
	return y, x
}
