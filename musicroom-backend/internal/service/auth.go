package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nfauconn/music-room/musicroom-backend/internal/auth"
	"github.com/nfauconn/music-room/musicroom-backend/internal/mail"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account temporarily locked")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrLastLoginMethod    = errors.New("cannot remove the only login method")
	ErrOAuthConflict      = errors.New("this account is already linked to another user")
	ErrOAuthUnavailable   = errors.New("google sign-in is not configured")
)

const (
	maxFailedLogins = 5
	lockoutWindow   = 15 * time.Minute
	lockoutDuration = 15 * time.Minute
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type ReqMeta struct {
	UserAgent *string
	IP        *netip.Addr
}

type AuthService struct {
	pool       *pgxpool.Pool
	q          *repository.Queries
	jwt        *auth.JWTManager
	mailer     mail.Mailer
	google     auth.GoogleVerifier
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(pool *pgxpool.Pool, jwtMgr *auth.JWTManager, mailer mail.Mailer, google auth.GoogleVerifier, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		pool:       pool,
		q:          repository.New(pool),
		jwt:        jwtMgr,
		mailer:     mailer,
		google:     google,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (repository.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return repository.User{}, err
	}
	user, err := s.q.CreateUser(ctx, repository.CreateUserParams{
		Email:         email,
		PasswordHash:  &hash,
		DisplayName:   displayName,
		EmailVerified: false,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repository.User{}, ErrEmailTaken
		}
		return repository.User{}, err
	}
	if err := s.sendEmailVerification(ctx, user); err != nil {
		log.Printf("mail: verification send failed for %s: %v", user.Email, err)
	}
	return user, nil
}

func (s *AuthService) sendEmailVerification(ctx context.Context, user repository.User) error {
	token, hash, err := auth.GenerateToken()
	if err != nil {
		return err
	}
	if _, err := s.q.CreateEmailVerification(ctx, repository.CreateEmailVerificationParams{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: ts(time.Now().Add(24 * time.Hour)),
	}); err != nil {
		return err
	}
	body := fmt.Sprintf("Welcome to MusicRoom!\n\nVerify your address with this token:\n%s\n\nExpires in 24h.", token)
	return s.mailer.Send(user.Email, "MusicRoom — verify your email address", body)
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	userID, err := s.q.ConsumeEmailVerification(ctx, auth.HashToken(token))
	if err != nil {
		return ErrInvalidToken
	}
	return s.q.SetEmailVerified(ctx, userID)
}

func (s *AuthService) Login(ctx context.Context, email, password string, meta ReqMeta) (TokenPair, repository.User, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, repository.User{}, ErrInvalidCredentials
		}
		return TokenPair{}, repository.User{}, err
	}
	now := time.Now()
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(now) {
		return TokenPair{}, repository.User{}, ErrAccountLocked
	}
	if user.PasswordHash == nil || !auth.CheckPassword(*user.PasswordHash, password) {
		s.registerFailedLogin(ctx, user, now)
		return TokenPair{}, repository.User{}, ErrInvalidCredentials
	}
	_ = s.q.ResetFailedLogin(ctx, user.ID)

	if !user.EmailVerified {
		return TokenPair{}, repository.User{}, ErrEmailNotVerified
	}

	pair, err := s.issueTokens(ctx, user.ID, meta)
	if err != nil {
		return TokenPair{}, repository.User{}, err
	}
	return pair, user, nil
}

func (s *AuthService) IssueTokensFor(ctx context.Context, userID uuid.UUID, meta ReqMeta) (TokenPair, error) {
	return s.issueTokens(ctx, userID, meta)
}

func (s *AuthService) registerFailedLogin(ctx context.Context, user repository.User, now time.Time) {
	if user.LastFailedLoginAt.Valid && now.Sub(user.LastFailedLoginAt.Time) > lockoutWindow {
		_ = s.q.ResetFailedLogin(ctx, user.ID)
	}
	row, err := s.q.IncrementFailedLogin(ctx, user.ID)
	if err != nil {
		return
	}
	if row.FailedLoginCount >= maxFailedLogins {
		_ = s.q.LockUser(ctx, repository.LockUserParams{
			ID:          user.ID,
			LockedUntil: ts(now.Add(lockoutDuration)),
		})
	}
}

func (s *AuthService) issueTokens(ctx context.Context, userID uuid.UUID, meta ReqMeta) (TokenPair, error) {
	epoch, err := s.q.GetTokenEpoch(ctx, userID)
	if err != nil {
		return TokenPair{}, err
	}
	access, err := s.jwt.GenerateAccess(userID, epoch)
	if err != nil {
		return TokenPair{}, err
	}
	token, hash, err := auth.GenerateToken()
	if err != nil {
		return TokenPair{}, err
	}
	if _, err := s.q.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: hash,
		FamilyID:  uuid.New(),
		ExpiresAt: ts(time.Now().Add(s.refreshTTL)),
		UserAgent: meta.UserAgent,
		IpAddress: meta.IP,
	}); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: token, ExpiresIn: int(s.accessTTL.Seconds())}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string, meta ReqMeta) (TokenPair, error) {
	rt, err := s.q.GetRefreshTokenByHash(ctx, auth.HashToken(refreshToken))
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}
	if rt.RevokedAt.Valid {
		_ = s.q.RevokeTokenFamily(ctx, rt.FamilyID)
		return TokenPair{}, ErrInvalidToken
	}
	if !rt.ExpiresAt.Valid || rt.ExpiresAt.Time.Before(time.Now()) {
		return TokenPair{}, ErrInvalidToken
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	newToken, newHash, err := auth.GenerateToken()
	if err != nil {
		return TokenPair{}, err
	}
	newRT, err := qtx.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    rt.UserID,
		TokenHash: newHash,
		FamilyID:  rt.FamilyID,
		ExpiresAt: ts(time.Now().Add(s.refreshTTL)),
		UserAgent: meta.UserAgent,
		IpAddress: meta.IP,
	})
	if err != nil {
		return TokenPair{}, err
	}
	if err := qtx.RotateRefreshToken(ctx, repository.RotateRefreshTokenParams{
		ID:         rt.ID,
		ReplacedBy: pgtype.UUID{Bytes: newRT.ID, Valid: true},
	}); err != nil {
		return TokenPair{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TokenPair{}, err
	}

	epoch, err := s.q.GetTokenEpoch(ctx, rt.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	access, err := s.jwt.GenerateAccess(rt.UserID, epoch)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: access, RefreshToken: newToken, ExpiresIn: int(s.accessTTL.Seconds())}, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil
	}
	token, hash, err := auth.GenerateToken()
	if err != nil {
		return nil
	}
	if _, err := s.q.CreatePasswordReset(ctx, repository.CreatePasswordResetParams{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: ts(time.Now().Add(time.Hour)),
	}); err != nil {
		return nil
	}
	body := fmt.Sprintf("Reset your password using this token:\n%s\n\nExpires in 1h.", token)
	if err := s.mailer.Send(user.Email, "MusicRoom — password reset", body); err != nil {
		log.Printf("mail: password reset send failed for %s: %v", user.Email, err)
	}
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	userID, err := s.q.ConsumePasswordReset(ctx, auth.HashToken(token))
	if err != nil {
		return ErrInvalidToken
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if err := qtx.SetPassword(ctx, repository.SetPasswordParams{ID: userID, PasswordHash: &hash}); err != nil {
		return err
	}
	if err := qtx.BumpTokenEpoch(ctx, userID); err != nil {
		return err
	}
	if err := qtx.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AuthService) VerifyAccessToken(ctx context.Context, claims auth.AccessClaims) error {
	row, err := s.q.CheckAccessToken(ctx, repository.CheckAccessTokenParams{
		ID:  claims.UserID,
		Jti: claims.JTI,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.ErrTokenRevoked
		}
		return err
	}
	if row.Revoked || row.TokenEpoch != claims.Epoch {
		return auth.ErrTokenRevoked
	}
	return nil
}

func (s *AuthService) Logout(ctx context.Context, claims auth.AccessClaims, refreshToken string) error {
	if err := s.q.RevokeAccessToken(ctx, repository.RevokeAccessTokenParams{
		Jti:       claims.JTI,
		UserID:    claims.UserID,
		ExpiresAt: ts(claims.ExpiresAt),
	}); err != nil {
		return err
	}
	if refreshToken != "" {
		rt, err := s.q.GetRefreshTokenByHash(ctx, auth.HashToken(refreshToken))
		if err == nil && rt.UserID == claims.UserID {
			if err := s.q.RevokeTokenFamily(ctx, rt.FamilyID); err != nil {
				return err
			}
		}
	}
	if err := s.q.PurgeExpiredRevokedTokens(ctx); err != nil {
		log.Printf("auth: purging expired revoked access tokens failed: %v", err)
	}
	return nil
}

func (s *AuthService) OAuthGoogle(ctx context.Context, idToken string, meta ReqMeta) (TokenPair, repository.User, error) {
	claims, err := s.google.Verify(ctx, idToken)
	if err != nil {
		if errors.Is(err, auth.ErrNotConfigured) {
			return TokenPair{}, repository.User{}, ErrOAuthUnavailable
		}
		return TokenPair{}, repository.User{}, ErrInvalidToken
	}

	acc, err := s.q.GetOAuthAccount(ctx, repository.GetOAuthAccountParams{
		Provider:       repository.OauthProviderGoogle,
		ProviderUserID: claims.Sub,
	})
	if err == nil {
		user, err := s.q.GetUserByID(ctx, acc.UserID)
		if err != nil {
			return TokenPair{}, repository.User{}, err
		}
		return s.finishOAuth(ctx, user, meta)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return TokenPair{}, repository.User{}, err
	}

	if claims.Email != "" {
		user, err := s.q.GetUserByEmail(ctx, claims.Email)
		if err == nil {
			if _, err := s.q.LinkOAuthAccount(ctx, repository.LinkOAuthAccountParams{
				UserID:         user.ID,
				Provider:       repository.OauthProviderGoogle,
				ProviderUserID: claims.Sub,
			}); err != nil {
				return TokenPair{}, repository.User{}, err
			}
			return s.finishOAuth(ctx, user, meta)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, repository.User{}, err
		}
	}

	user, err := s.q.CreateUser(ctx, repository.CreateUserParams{
		Email:         claims.Email,
		PasswordHash:  nil,
		DisplayName:   displayNameFrom(claims),
		EmailVerified: claims.EmailVerified,
	})
	if err != nil {
		return TokenPair{}, repository.User{}, err
	}
	if _, err := s.q.LinkOAuthAccount(ctx, repository.LinkOAuthAccountParams{
		UserID:         user.ID,
		Provider:       repository.OauthProviderGoogle,
		ProviderUserID: claims.Sub,
	}); err != nil {
		return TokenPair{}, repository.User{}, err
	}
	return s.finishOAuth(ctx, user, meta)
}

func (s *AuthService) finishOAuth(ctx context.Context, user repository.User, meta ReqMeta) (TokenPair, repository.User, error) {
	pair, err := s.issueTokens(ctx, user.ID, meta)
	if err != nil {
		return TokenPair{}, repository.User{}, err
	}
	return pair, user, nil
}

func (s *AuthService) LinkGoogle(ctx context.Context, userID uuid.UUID, idToken string) error {
	claims, err := s.google.Verify(ctx, idToken)
	if err != nil {
		if errors.Is(err, auth.ErrNotConfigured) {
			return ErrOAuthUnavailable
		}
		return ErrInvalidToken
	}
	acc, err := s.q.GetOAuthAccount(ctx, repository.GetOAuthAccountParams{
		Provider:       repository.OauthProviderGoogle,
		ProviderUserID: claims.Sub,
	})
	if err == nil {
		if acc.UserID == userID {
			return nil
		}
		return ErrOAuthConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err := s.q.LinkOAuthAccount(ctx, repository.LinkOAuthAccountParams{
		UserID:         userID,
		Provider:       repository.OauthProviderGoogle,
		ProviderUserID: claims.Sub,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrOAuthConflict
		}
		return err
	}
	return nil
}

func (s *AuthService) UnlinkGoogle(ctx context.Context, userID uuid.UUID) error {
	methods, err := s.q.CountUserLoginMethods(ctx, userID)
	if err != nil {
		return err
	}
	hasPassword, _ := methods.HasPassword.(bool)
	if !hasPassword && methods.OauthCount <= 1 {
		return ErrLastLoginMethod
	}
	_, err = s.q.UnlinkOAuthAccount(ctx, repository.UnlinkOAuthAccountParams{
		UserID:   userID,
		Provider: repository.OauthProviderGoogle,
	})
	return err
}

func displayNameFrom(c *auth.GoogleClaims) string {
	if c.Name != "" {
		return c.Name
	}
	if i := strings.IndexByte(c.Email, '@'); i > 0 {
		return c.Email[:i]
	}
	return "MusicRoom User"
}
