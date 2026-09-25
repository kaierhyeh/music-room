package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler { return &UserHandler{svc: svc} }

type updateProfileRequest struct {
	ProfileData       json.RawMessage `json:"profile_data"`
	ProfileVisibility json.RawMessage `json:"profile_visibility"`
	MusicPreferences  json.RawMessage `json:"music_preferences"`
}

// @Summary     Get the authenticated user's full profile
// @Tags        users
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} map[string]any
// @Router      /users/me [get]
func (h *UserHandler) Me(c echo.Context) error {
	uid := mw.UserID(c)
	v, err := h.svc.GetProfile(c.Request().Context(), uid, uid)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, v)
}

// @Summary     Partially update the authenticated user's profile
// @Tags        users
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body updateProfileRequest true "Fields to update (omit to keep unchanged)"
// @Success     200 {object} map[string]any
// @Failure     400 {object} map[string]string
// @Router      /users/me [patch]
func (h *UserHandler) UpdateMe(c echo.Context) error {
	var req updateProfileRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}
	v, err := h.svc.UpdateProfile(c.Request().Context(), mw.UserID(c),
		req.ProfileData, req.ProfileVisibility, req.MusicPreferences)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, v)
}

// @Summary     Get a user's profile (filtered by friendship level)
// @Tags        users
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "User UUID"
// @Success     200 {object} map[string]any
// @Failure     404 {object} map[string]string
// @Router      /users/{id} [get]
func (h *UserHandler) GetByID(c echo.Context) error {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}
	v, err := h.svc.GetProfile(c.Request().Context(), mw.UserID(c), targetID)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, v)
}
