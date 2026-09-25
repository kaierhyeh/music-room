package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/nfauconn/music-room/musicroom-backend/internal/auth"
)

const (
	userIDKey = "userID"
	claimsKey = "accessClaims"
)

type AccessVerifier interface {
	VerifyAccessToken(ctx context.Context, claims auth.AccessClaims) error
}

func JWT(jwtMgr *auth.JWTManager, verifier AccessVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}
			claims, err := jwtMgr.ParseAccess(strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			if err := verifier.VerifyAccessToken(c.Request().Context(), claims); err != nil {
				if errors.Is(err, auth.ErrTokenRevoked) {
					return echo.NewHTTPError(http.StatusUnauthorized, "token revoked")
				}
				return echo.NewHTTPError(http.StatusServiceUnavailable, "cannot verify token")
			}
			c.Set(userIDKey, claims.UserID)
			c.Set(claimsKey, claims)
			return next(c)
		}
	}
}

func UserID(c echo.Context) uuid.UUID {
	id, _ := c.Get(userIDKey).(uuid.UUID)
	return id
}

func AccessToken(c echo.Context) auth.AccessClaims {
	claims, _ := c.Get(claimsKey).(auth.AccessClaims)
	return claims
}
