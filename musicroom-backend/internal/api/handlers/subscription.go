package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

type setSubscriptionRequest struct {
	Tier string `json:"tier" validate:"required,oneof=free premium"`
	Note string `json:"note" validate:"max=200"`
}

// @Summary     Read the caller's subscription
// @Description Returns the tier in force — a premium whose term has run out reads as free — and, for an active premium, its expiry.
// @Tags        subscription
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} service.Subscription
// @Failure     401 {object} map[string]string
// @Router      /users/me/subscription [get]
func (h *SubscriptionHandler) Get(c echo.Context) error {
	sub, err := h.svc.Get(c.Request().Context(), mw.UserID(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, sub)
}

// @Summary     Switch between the free and premium offers
// @Description Upgrading starts a 30-day term; downgrading takes effect at once. Both are recorded in the history.
// @Tags        subscription
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body setSubscriptionRequest true "Offer to switch to"
// @Success     200 {object} service.Subscription
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /users/me/subscription [put]
func (h *SubscriptionHandler) Set(c echo.Context) error {
	var req setSubscriptionRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	sub, err := h.svc.Set(c.Request().Context(), mw.UserID(c), req.Tier, req.Note)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, sub)
}

// @Summary     List the caller's subscription changes
// @Tags        subscription
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} service.HistoryEntry
// @Failure     401 {object} map[string]string
// @Router      /users/me/subscription/history [get]
func (h *SubscriptionHandler) History(c echo.Context) error {
	entries, err := h.svc.History(c.Request().Context(), mw.UserID(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, entries)
}
