package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrTokenRevoked = errors.New("access token revoked")

type JWTManager struct {
	secret    []byte
	accessTTL time.Duration
}

func NewJWTManager(secret string, accessTTL time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), accessTTL: accessTTL}
}

type AccessClaims struct {
	UserID    uuid.UUID
	JTI       uuid.UUID
	Epoch     int32
	ExpiresAt time.Time
}

type accessClaims struct {
	jwt.RegisteredClaims
	Epoch int32 `json:"epoch"`
}

func (m *JWTManager) GenerateAccess(userID uuid.UUID, epoch int32) (string, error) {
	now := time.Now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
		Epoch: epoch,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

func (m *JWTManager) ParseAccess(tokenStr string) (AccessClaims, error) {
	claims := &accessClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return AccessClaims{}, err
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return AccessClaims{}, fmt.Errorf("invalid subject: %w", err)
	}
	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return AccessClaims{}, fmt.Errorf("invalid jti: %w", err)
	}
	if claims.ExpiresAt == nil {
		return AccessClaims{}, fmt.Errorf("missing exp")
	}
	return AccessClaims{
		UserID:    id,
		JTI:       jti,
		Epoch:     claims.Epoch,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
