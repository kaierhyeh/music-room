package handlers

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/auth"
	"github.com/nfauconn/music-room/musicroom-backend/internal/hub"
)

type RoomGuard func(ctx context.Context, roomID, userID uuid.UUID) (bool, error)

type WSHandler struct {
	hub           *hub.Hub
	tickets       *auth.TicketStore
	upgrader      websocket.Upgrader
	eventGuard    RoomGuard
	playlistGuard RoomGuard
}

func NewWSHandler(h *hub.Hub, tickets *auth.TicketStore, allowedOrigins []string, eventGuard, playlistGuard RoomGuard) *WSHandler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &WSHandler{
		hub:           h,
		tickets:       tickets,
		eventGuard:    eventGuard,
		playlistGuard: playlistGuard,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				if u, err := url.Parse(origin); err == nil && u.Host == r.Host {
					return true
				}
				if len(allowed) == 0 {
					return true
				}
				return allowed[origin]
			},
		},
	}
}

// @Summary     Issue a one-time WebSocket ticket (30s TTL)
// @Tags        websocket
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} map[string]any
// @Router      /ws-tickets [post]
func (h *WSHandler) IssueTicket(c echo.Context) error {
	tok, err := h.tickets.Issue(mw.UserID(c), mw.Client(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ticket error")
	}
	return c.JSON(http.StatusOK, map[string]any{"ticket": tok, "expires_in": 30})
}

// @Summary     Open a WebSocket connection to an event room
// @Tags        websocket
// @Param       id     path  string true "Event UUID"
// @Param       ticket query string true "One-time ticket from POST /ws-tickets"
// @Success     101
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Router      /ws/events/{id} [get]
func (h *WSHandler) ConnectEvent(c echo.Context) error {
	return h.connect(c, h.eventGuard)
}

// @Summary     Open a WebSocket connection to a playlist room
// @Tags        websocket
// @Param       id     path  string true "Playlist UUID"
// @Param       ticket query string true "One-time ticket from POST /ws-tickets"
// @Success     101
// @Failure     401 {object} map[string]string
// @Failure     403 {object} map[string]string
// @Router      /ws/playlists/{id} [get]
func (h *WSHandler) ConnectPlaylist(c echo.Context) error {
	return h.connect(c, h.playlistGuard)
}

func (h *WSHandler) connect(c echo.Context, guard RoomGuard) error {
	room, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid room id")
	}
	userID, client, ok := h.tickets.Consume(c.QueryParam("ticket"))
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired ticket")
	}
	mw.Identify(c, userID, client)
	allowed, err := guard(c.Request().Context(), room, userID)
	if err != nil || !allowed {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return nil
	}
	h.hub.Add(room, conn)
	return nil
}
