package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type EventHandler struct {
	svc *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler { return &EventHandler{svc: svc} }

type createEventRequest struct {
	Name             string     `json:"name" validate:"required,min=1,max=100"`
	Description      string     `json:"description" validate:"max=500"`
	Visibility       string     `json:"visibility" validate:"omitempty,oneof=public private"`
	License          string     `json:"license" validate:"omitempty,oneof=open invited_only time_and_location"`
	VoteStartAt      *time.Time `json:"vote_start_at"`
	VoteEndAt        *time.Time `json:"vote_end_at"`
	VoteLatitude     *float64   `json:"vote_latitude"`
	VoteLongitude    *float64   `json:"vote_longitude"`
	VoteRadiusMeters *int       `json:"vote_radius_meters"`
}

// @Summary     Create an event
// @Tags        events
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body createEventRequest true "Event payload"
// @Success     201 {object} map[string]any
// @Failure     400 {object} map[string]string
// @Router      /events [post]
func (h *EventHandler) Create(c echo.Context) error {
	var req createEventRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	vis := repository.VisibilityPublic
	if req.Visibility == "private" {
		vis = repository.VisibilityPrivate
	}
	lic := repository.EventLicenseOpen
	switch req.License {
	case "invited_only":
		lic = repository.EventLicenseInvitedOnly
	case "time_and_location":
		lic = repository.EventLicenseTimeAndLocation
	}
	ev, err := h.svc.CreateEvent(c.Request().Context(), service.CreateEventInput{
		OwnerID:          mw.UserID(c),
		Name:             req.Name,
		Description:      req.Description,
		Visibility:       vis,
		License:          lic,
		VoteStartAt:      req.VoteStartAt,
		VoteEndAt:        req.VoteEndAt,
		VoteLatitude:     req.VoteLatitude,
		VoteLongitude:    req.VoteLongitude,
		VoteRadiusMeters: req.VoteRadiusMeters,
	})
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusCreated, eventJSON(ev))
}

// @Summary     List the events the caller may open
// @Description Public events, plus the private ones the caller hosts or was invited to.
// @Tags        events
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} map[string]any
// @Router      /events [get]
func (h *EventHandler) List(c echo.Context) error {
	rows, err := h.svc.ListVisible(c.Request().Context(), mw.UserID(c), 50, 0)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

// @Summary     Get event details with sorted track queue
// @Tags        events
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "Event UUID"
// @Success     200 {object} map[string]any
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /events/{id} [get]
func (h *EventHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}
	ev, ownerName, queue, err := h.svc.GetEvent(c.Request().Context(), mw.UserID(c), id)
	if err != nil {
		return mapErr(err)
	}
	resp := eventJSON(ev)
	resp["owner_display_name"] = ownerName
	resp["queue"] = queue
	return c.JSON(http.StatusOK, resp)
}

type suggestTrackRequest struct {
	DeezerID int64 `json:"deezer_id" validate:"required"`
}

// @Summary     Suggest a track for an event
// @Tags        events
// @Security    BearerAuth
// @Accept      json
// @Param       id path string true "Event UUID"
// @Param       body body suggestTrackRequest true "Deezer track ID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /events/{id}/tracks [post]
func (h *EventHandler) SuggestTrack(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}
	var req suggestTrackRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.SuggestTrack(c.Request().Context(), id, mw.UserID(c), req.DeezerID); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

type voteRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// @Summary     Cast a vote on a track (idempotent)
// @Tags        events
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id      path string   true  "Event UUID"
// @Param       trackId path string   true  "EventTrack UUID"
// @Param       body    body voteRequest false "GPS coords (required for time_and_location)"
// @Success     200 {object} map[string]bool
// @Failure     403 {object} map[string]string
// @Router      /events/{id}/tracks/{trackId}/vote [post]
func (h *EventHandler) Vote(c echo.Context) error {
	eventID, trackID, err := eventTrackParams(c)
	if err != nil {
		return err
	}
	var vr voteRequest
	_ = c.Bind(&vr)
	changed, err := h.svc.CastVote(c.Request().Context(), eventID, mw.UserID(c), trackID, vr.Latitude, vr.Longitude)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"voted": changed})
}

// @Summary     Remove vote from a track
// @Tags        events
// @Security    BearerAuth
// @Produce     json
// @Param       id      path string true "Event UUID"
// @Param       trackId path string true "EventTrack UUID"
// @Success     200 {object} map[string]bool
// @Failure     403 {object} map[string]string
// @Router      /events/{id}/tracks/{trackId}/vote [delete]
func (h *EventHandler) Unvote(c echo.Context) error {
	eventID, trackID, err := eventTrackParams(c)
	if err != nil {
		return err
	}
	changed, err := h.svc.RemoveVote(c.Request().Context(), eventID, mw.UserID(c), trackID)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"removed": changed})
}

type inviteRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// @Summary     Invite a user to an event (owner only)
// @Tags        events
// @Security    BearerAuth
// @Accept      json
// @Param       id   path string      true "Event UUID"
// @Param       body body inviteRequest true "User to invite"
// @Success     204
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /events/{id}/invitations [post]
func (h *EventHandler) Invite(c echo.Context) error {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}
	var req inviteRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	inviteeID, _ := uuid.Parse(req.UserID)
	if err := h.svc.InviteToEvent(c.Request().Context(), mw.UserID(c), eventID, inviteeID); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     List invitations for an event (owner only)
// @Tags        events
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "Event UUID"
// @Success     200 {array} map[string]any
// @Failure     403 {object} map[string]string
// @Router      /events/{id}/invitations [get]
func (h *EventHandler) ListInvitations(c echo.Context) error {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}
	rows, err := h.svc.ListInvitations(c.Request().Context(), mw.UserID(c), eventID)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

func eventTrackParams(c echo.Context) (uuid.UUID, uuid.UUID, error) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}
	trackID, err := uuid.Parse(c.Param("trackId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid track id")
	}
	return eventID, trackID, nil
}

func eventJSON(ev repository.Event) map[string]any {
	return map[string]any{
		"id":                 ev.ID,
		"owner_id":           ev.OwnerID,
		"name":               ev.Name,
		"description":        ev.Description,
		"visibility":         ev.Visibility,
		"license":            ev.License,
		"vote_start_at":      ev.VoteStartAt,
		"vote_end_at":        ev.VoteEndAt,
		"vote_latitude":      ev.VoteLatitude,
		"vote_longitude":     ev.VoteLongitude,
		"vote_radius_meters": ev.VoteRadiusMeters,
		"created_at":         ev.CreatedAt,
	}
}
