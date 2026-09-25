package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type FriendHandler struct {
	svc *service.FriendService
}

func NewFriendHandler(svc *service.FriendService) *FriendHandler { return &FriendHandler{svc: svc} }

type friendRequestBody struct {
	AddresseeID string `json:"addressee_id" validate:"required,uuid"`
}

// @Summary     Send a friend request (auto-accepts if reverse request exists)
// @Tags        friends
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body friendRequestBody true "Target user ID"
// @Success     201 {object} map[string]string
// @Failure     400 {object} map[string]string
// @Failure     409 {object} map[string]string
// @Router      /friends/requests [post]
func (h *FriendHandler) Send(c echo.Context) error {
	var req friendRequestBody
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	addr, _ := uuid.Parse(req.AddresseeID)
	status, err := h.svc.SendRequest(c.Request().Context(), mw.UserID(c), addr)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": status})
}

// @Summary     Accept a friend request
// @Tags        friends
// @Security    BearerAuth
// @Param       id path string true "Request UUID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /friends/requests/{id}/accept [post]
func (h *FriendHandler) Accept(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request id")
	}
	if err := h.svc.Accept(c.Request().Context(), mw.UserID(c), id); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     Reject a friend request
// @Tags        friends
// @Security    BearerAuth
// @Param       id path string true "Request UUID"
// @Success     204
// @Failure     403 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /friends/requests/{id}/reject [post]
func (h *FriendHandler) Reject(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request id")
	}
	if err := h.svc.Reject(c.Request().Context(), mw.UserID(c), id); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     List accepted friends
// @Tags        friends
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} map[string]any
// @Router      /friends [get]
func (h *FriendHandler) List(c echo.Context) error {
	rows, err := h.svc.ListFriends(c.Request().Context(), mw.UserID(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}

// @Summary     List incoming pending friend requests
// @Tags        friends
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} map[string]any
// @Router      /friends/requests [get]
func (h *FriendHandler) ListIncoming(c echo.Context) error {
	rows, err := h.svc.ListIncoming(c.Request().Context(), mw.UserID(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, rows)
}
