package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type TrackHandler struct {
	svc *service.TrackService
}

func NewTrackHandler(svc *service.TrackService) *TrackHandler { return &TrackHandler{svc: svc} }

// @Summary     Search tracks via Deezer
// @Tags        tracks
// @Security    BearerAuth
// @Produce     json
// @Param       q query string true "Search query"
// @Success     200 {array} map[string]any
// @Failure     400 {object} map[string]string
// @Failure     502 {object} map[string]string
// @Router      /tracks/search [get]
func (h *TrackHandler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter q is required")
	}
	tracks, err := h.svc.Search(c.Request().Context(), q, 25)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "Deezer search unavailable")
	}
	return c.JSON(http.StatusOK, tracks)
}

// @Summary     Get a playable preview URL for a track
// @Description Deezer preview URLs are signed and expire, so the one stored with a playlist or event track goes stale. This re-reads it from Deezer.
// @Tags        tracks
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "Deezer track id"
// @Success     200 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /tracks/{id}/preview [get]
func (h *TrackHandler) Preview(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid track id")
	}
	url, err := h.svc.FreshPreview(c.Request().Context(), id)
	if err != nil {
		return mapErr(err)
	}
	if url == "" {
		return echo.NewHTTPError(http.StatusNotFound, "no preview for this track")
	}
	return c.JSON(http.StatusOK, map[string]string{"preview_url": url})
}
