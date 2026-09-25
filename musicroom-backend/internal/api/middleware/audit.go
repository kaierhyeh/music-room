package middleware

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/nfauconn/music-room/musicroom-backend/internal/audit"
	"github.com/nfauconn/music-room/musicroom-backend/internal/auth"
)

const clientKey = "auditClient"

func Client(c echo.Context) auth.Client {
	if cl, ok := c.Get(clientKey).(auth.Client); ok {
		return cl
	}
	h := c.Request().Header
	return auth.Client{
		Platform:    h.Get("X-Platform"),
		DeviceModel: h.Get("X-Device-Model"),
		AppVersion:  h.Get("X-App-Version"),
	}
}

func Identify(c echo.Context, userID uuid.UUID, client auth.Client) {
	c.Set(userIDKey, userID)
	c.Set(clientKey, client)
}

func Audit(rec *audit.Recorder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			if c.Path() == "/health" {
				return err
			}

			status := c.Response().Status
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				} else {
					status = http.StatusInternalServerError
				}
			}

			var ip *netip.Addr
			if a, e := netip.ParseAddr(c.RealIP()); e == nil {
				ip = &a
			}

			resType, resID := resource(c)
			client := Client(c)

			rec.Record(audit.Entry{
				UserID:       UserID(c),
				Action:       c.Request().Method + " " + c.Path(),
				ResourceType: resType,
				ResourceID:   resID,
				Platform:     client.Platform,
				DeviceModel:  client.DeviceModel,
				AppVersion:   client.AppVersion,
				IP:           ip,
				UserAgent:    c.Request().UserAgent(),
				StatusCode:   status,
			})
			return err
		}
	}
}

func resource(c echo.Context) (string, string) {
	names := c.ParamNames()
	if len(names) == 0 {
		return "", ""
	}
	deepest := names[len(names)-1]
	id := c.Param(deepest)
	if id == "" {
		return "", ""
	}

	segments := strings.Split(strings.Trim(c.Path(), "/"), "/")
	collection := ""
	for i, seg := range segments {
		if seg == ":"+deepest && i > 0 {
			collection = segments[i-1]
			break
		}
	}
	if collection == "" {
		return "", id
	}
	return singular(collection), id
}

func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies"):
		return strings.TrimSuffix(s, "ies") + "y"
	case strings.HasSuffix(s, "s"):
		return strings.TrimSuffix(s, "s")
	}
	return s
}
