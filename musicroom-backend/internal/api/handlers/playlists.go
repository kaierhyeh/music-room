package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type PlaylistHandler struct {
	svc *service.PlaylistService
}

func NewPlaylistHandler(svc *service.PlaylistService) *PlaylistHandler {
	return &PlaylistHandler{svc: svc}
}

type createPlaylistRequest struct {
	Name          string                `json:"name" validate:"required,min=1,max=100"`
	Description   string                `json:"description" validate:"max=500"`
	Visibility    string                `json:"visibility" validate:"omitempty,oneof=public private"`
	License       string                `json:"license" validate:"omitempty,oneof=open invited_only"`
	Collaborators []collaboratorRequest `json:"collaborators" validate:"max=100,dive"`
}

// @Summary     Create a playlist
// @Tags        playlists
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body createPlaylistRequest true "Playlist payload"
// @Success     201 {object} map[string]any
// @Failure     400 {object} map[string]string
// @Router      /playlists [post]
func (h *PlaylistHandler) Create(c echo.Context) error {
	var req createPlaylistRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	vis := repository.VisibilityPublic
	if req.Visibility == "private" {
		vis = repository.VisibilityPrivate
	}
	lic := repository.PlaylistLicenseOpen
	if req.License == "invited_only" {
		lic = repository.PlaylistLicenseInvitedOnly
	}
	invites := make([]service.Invite, 0, len(req.Collaborators))
	for _, co := range req.Collaborators {
		uid, _ := uuid.Parse(co.UserID)
		invites = append(invites, service.Invite{UserID: uid, CanEdit: co.CanEdit == nil || *co.CanEdit})
	}
	pl, ownerName, err := h.svc.Create(c.Request().Context(), mw.UserID(c), req.Name, req.Description, vis, lic, invites)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusCreated, playlistJSON(pl, ownerName))
}

type updatePlaylistRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Visibility  *string `json:"visibility" validate:"omitempty,oneof=public private"`
	License     *string `json:"license" validate:"omitempty,oneof=open invited_only"`
}

// @Summary     Update a playlist's metadata (owner only)
// @Description Every field is optional; an omitted one keeps its current value.
// @Tags        playlists
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id   path string                 true "Playlist UUID"
// @Param       body body updatePlaylistRequest  true "Fields to change"
// @Success     200 {object} map[string]any
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /playlists/{id} [patch]
func (h *PlaylistHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	var req updatePlaylistRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	upd := service.PlaylistUpdate{Name: req.Name, Description: req.Description}
	if req.Visibility != nil {
		vis := repository.VisibilityPublic
		if *req.Visibility == "private" {
			vis = repository.VisibilityPrivate
		}
		upd.Visibility = &vis
	}
	if req.License != nil {
		lic := repository.PlaylistLicenseOpen
		if *req.License == "invited_only" {
			lic = repository.PlaylistLicenseInvitedOnly
		}
		upd.License = &lic
	}
	pl, ownerName, err := h.svc.Update(c.Request().Context(), mw.UserID(c), id, upd)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, playlistJSON(pl, ownerName))
}

// @Summary     List public playlists
// @Tags        playlists
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} map[string]any
// @Router      /playlists [get]
func (h *PlaylistHandler) List(c echo.Context) error {
	rows, err := h.svc.ListPublic(c.Request().Context(), 50, 0)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

// @Summary     List the playlists the caller created
// @Description Private playlists included — the owner always sees their own.
// @Tags        playlists
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} map[string]any
// @Router      /playlists/mine [get]
func (h *PlaylistHandler) ListMine(c echo.Context) error {
	rows, err := h.svc.ListMine(c.Request().Context(), mw.UserID(c), 50, 0)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

// @Summary     List the playlists the caller was invited onto
// @Description Private playlists included. Each row carries can_edit: whether the invitation is to edit or only to view.
// @Tags        playlists
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} map[string]any
// @Router      /playlists/shared [get]
func (h *PlaylistHandler) ListShared(c echo.Context) error {
	rows, err := h.svc.ListShared(c.Request().Context(), mw.UserID(c), 50, 0)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

// @Summary     Get playlist with ordered tracks
// @Tags        playlists
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "Playlist UUID"
// @Success     200 {object} map[string]any
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /playlists/{id} [get]
func (h *PlaylistHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	pl, ownerName, tracks, err := h.svc.Get(c.Request().Context(), mw.UserID(c), id)
	if err != nil {
		return mapErr(err)
	}
	resp := playlistJSON(pl, ownerName)
	resp["tracks"] = tracks
	return c.JSON(http.StatusOK, resp)
}

type addPlaylistTrackRequest struct {
	DeezerID     int64   `json:"deezer_id" validate:"required"`
	AfterTrackID *string `json:"after_track_id" validate:"omitempty,uuid"`
}

// @Summary     Add a track to a playlist at a given position
// @Tags        playlists
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id   path string              true "Playlist UUID"
// @Param       body body addPlaylistTrackRequest true "Track + optional anchor"
// @Success     201 {object} map[string]any
// @Failure     403 {object} map[string]string
// @Failure     409 {object} map[string]string
// @Router      /playlists/{id}/tracks [post]
func (h *PlaylistHandler) AddTrack(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	var req addPlaylistTrackRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	ver, err := h.svc.AddTrack(c.Request().Context(), mw.UserID(c), pid, req.DeezerID, parseOptUUID(req.AfterTrackID))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"version": ver})
}

type moveTrackRequest struct {
	AfterTrackID *string `json:"after_track_id" validate:"omitempty,uuid"`
	Version      *int32  `json:"version" validate:"required"`
}

// @Summary     Move a track (optimistic locking via version)
// @Tags        playlists
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id      path string          true "Playlist UUID"
// @Param       trackId path string          true "PlaylistTrack UUID"
// @Param       body    body moveTrackRequest true "Target anchor + expected version"
// @Success     200 {object} map[string]any
// @Failure     403 {object} map[string]string
// @Failure     409 {object} map[string]string
// @Router      /playlists/{id}/tracks/{trackId} [patch]
func (h *PlaylistHandler) MoveTrack(c echo.Context) error {
	pid, tid, err := playlistTrackParams(c)
	if err != nil {
		return err
	}
	var req moveTrackRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	ver, err := h.svc.MoveTrack(c.Request().Context(), mw.UserID(c), pid, tid, parseOptUUID(req.AfterTrackID), *req.Version)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, map[string]any{"version": ver})
}

// @Summary     Remove a track from a playlist
// @Tags        playlists
// @Security    BearerAuth
// @Produce     json
// @Param       id      path string true "Playlist UUID"
// @Param       trackId path string true "PlaylistTrack UUID"
// @Success     200 {object} map[string]any
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /playlists/{id}/tracks/{trackId} [delete]
func (h *PlaylistHandler) RemoveTrack(c echo.Context) error {
	pid, tid, err := playlistTrackParams(c)
	if err != nil {
		return err
	}
	ver, err := h.svc.RemoveTrack(c.Request().Context(), mw.UserID(c), pid, tid)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, map[string]any{"version": ver})
}

type collaboratorRequest struct {
	UserID  string `json:"user_id" validate:"required,uuid"`
	CanEdit *bool  `json:"can_edit"`
}

// @Summary     Invite a user onto a playlist, or change their right (owner only)
// @Description can_edit=false invites a viewer, who may open a private playlist but not edit it. Re-inviting someone updates their right.
// @Tags        playlists
// @Security    BearerAuth
// @Accept      json
// @Param       id   path string             true "Playlist UUID"
// @Param       body body collaboratorRequest true "User to add"
// @Success     204
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /playlists/{id}/collaborators [post]
func (h *PlaylistHandler) AddCollaborator(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	var req collaboratorRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	uid, _ := uuid.Parse(req.UserID)
	canEdit := req.CanEdit == nil || *req.CanEdit
	if err := h.svc.AddCollaborator(c.Request().Context(), mw.UserID(c), pid, uid, canEdit); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     Revoke a collaborator's invitation (owner only)
// @Tags        playlists
// @Security    BearerAuth
// @Param       id     path string true "Playlist UUID"
// @Param       userId path string true "Collaborator UUID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /playlists/{id}/collaborators/{userId} [delete]
func (h *PlaylistHandler) RemoveCollaborator(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	uid, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	if err := h.svc.RemoveCollaborator(c.Request().Context(), mw.UserID(c), pid, uid); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     List collaborators of a playlist
// @Tags        playlists
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "Playlist UUID"
// @Success     200 {array} map[string]any
// @Failure     403 {object} map[string]string
// @Router      /playlists/{id}/collaborators [get]
func (h *PlaylistHandler) ListCollaborators(c echo.Context) error {
	pid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	rows, err := h.svc.ListCollaborators(c.Request().Context(), mw.UserID(c), pid)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

func playlistTrackParams(c echo.Context) (uuid.UUID, uuid.UUID, error) {
	pid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	tid, err := uuid.Parse(c.Param("trackId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid track id")
	}
	return pid, tid, nil
}

func parseOptUUID(s *string) *uuid.UUID {
	if s == nil {
		return nil
	}
	u, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &u
}

func playlistJSON(pl repository.Playlist, ownerName string) map[string]any {
	return map[string]any{
		"id":                 pl.ID,
		"owner_id":           pl.OwnerID,
		"owner_display_name": ownerName,
		"name":               pl.Name,
		"description":        pl.Description,
		"visibility":         pl.Visibility,
		"license":            pl.License,
		"version":            pl.Version,
		"created_at":         pl.CreatedAt,
	}
}
