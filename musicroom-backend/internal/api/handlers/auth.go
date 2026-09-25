package handlers

import (
	"errors"
	"net/http"
	"net/netip"

	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

type registerRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8,max=72"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=50"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type tokenRequest struct {
	Token string `json:"token" validate:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type googleAuthRequest struct {
	IDToken string `json:"id_token" validate:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func toTokenResponse(p service.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    p.ExpiresIn,
	}
}

// @Summary     Register a new user
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body registerRequest true "Registration payload"
// @Success     201 {object} map[string]any
// @Failure     400 {object} map[string]string
// @Failure     409 {object} map[string]string
// @Router      /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req registerRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	user, err := h.svc.Register(c.Request().Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"id":             user.ID,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"email_verified": user.EmailVerified,
	})
}

// @Summary     Authenticate with email and password
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body loginRequest true "Credentials"
// @Success     200 {object} tokenResponse
// @Failure     401 {object} map[string]string
// @Failure     429 {object} map[string]string
// @Router      /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	pair, _, err := h.svc.Login(c.Request().Context(), req.Email, req.Password, reqMeta(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, toTokenResponse(pair))
}

// @Summary     Rotate refresh token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body refreshRequest true "Refresh token"
// @Success     200 {object} tokenResponse
// @Failure     400 {object} map[string]string
// @Router      /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	pair, err := h.svc.Refresh(c.Request().Context(), req.RefreshToken, reqMeta(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, toTokenResponse(pair))
}

// @Summary     Verify email address
// @Tags        auth
// @Accept      json
// @Param       body body tokenRequest true "Verification token"
// @Success     204
// @Failure     400 {object} map[string]string
// @Router      /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req tokenRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.VerifyEmail(c.Request().Context(), req.Token); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     Request a password reset email
// @Tags        auth
// @Accept      json
// @Param       body body forgotPasswordRequest true "Email address"
// @Success     202
// @Router      /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req forgotPasswordRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	_ = h.svc.ForgotPassword(c.Request().Context(), req.Email)
	return c.NoContent(http.StatusAccepted)
}

// @Summary     Apply a new password using reset token
// @Tags        auth
// @Accept      json
// @Param       body body resetPasswordRequest true "Token + new password"
// @Success     204
// @Failure     400 {object} map[string]string
// @Router      /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req resetPasswordRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.ResetPassword(c.Request().Context(), req.Token, req.Password); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     End the calling session
// @Description Revokes the access token used for this request and, if given, the refresh token and its rotation family. Other sessions of the same user are untouched.
// @Tags        auth
// @Accept      json
// @Security    BearerAuth
// @Param       body body logoutRequest false "Refresh token to revoke as well"
// @Success     204
// @Failure     401 {object} map[string]string
// @Router      /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	var req logoutRequest
	_ = c.Bind(&req)
	if err := h.svc.Logout(c.Request().Context(), mw.AccessToken(c), req.RefreshToken); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     Sign in or register with a Google id_token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body googleAuthRequest true "Google id_token"
// @Success     200 {object} tokenResponse
// @Failure     401 {object} map[string]string
// @Router      /auth/google [post]
func (h *AuthHandler) AuthGoogle(c echo.Context) error {
	var req googleAuthRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	pair, _, err := h.svc.OAuthGoogle(c.Request().Context(), req.IDToken, reqMeta(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, toTokenResponse(pair))
}

// @Summary     Link a Google account to the authenticated user
// @Tags        auth
// @Security    BearerAuth
// @Accept      json
// @Param       body body googleAuthRequest true "Google id_token"
// @Success     204
// @Failure     409 {object} map[string]string
// @Router      /users/me/oauth/google [post]
func (h *AuthHandler) LinkGoogle(c echo.Context) error {
	var req googleAuthRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.LinkGoogle(c.Request().Context(), mw.UserID(c), req.IDToken); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// @Summary     Unlink Google account (refused if it is the only login method)
// @Tags        auth
// @Security    BearerAuth
// @Success     204
// @Failure     409 {object} map[string]string
// @Router      /users/me/oauth/google [delete]
func (h *AuthHandler) UnlinkGoogle(c echo.Context) error {
	if err := h.svc.UnlinkGoogle(c.Request().Context(), mw.UserID(c)); err != nil {
		return mapErr(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func bindValidate(c echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}
	if err := c.Validate(v); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func reqMeta(c echo.Context) service.ReqMeta {
	var m service.ReqMeta
	if ua := c.Request().UserAgent(); ua != "" {
		m.UserAgent = &ua
	}
	if ip, err := netip.ParseAddr(c.RealIP()); err == nil {
		m.IP = &ip
	}
	return m
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrAccountLocked):
		return echo.NewHTTPError(http.StatusTooManyRequests, err.Error())
	case errors.Is(err, service.ErrEmailNotVerified):
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrEmailTaken):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidToken):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrLastLoginMethod):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrOAuthConflict):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrOAuthUnavailable):
		return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, service.ErrPremiumRequired):
		return echo.NewHTTPError(http.StatusPaymentRequired, err.Error())
	case errors.Is(err, service.ErrUnknownTier):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidProfile):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrCannotFriendSelf):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrAlreadyFriends), errors.Is(err, service.ErrRequestExists):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrRequestNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrNotAddressee), errors.Is(err, service.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrEventNotFound),
		errors.Is(err, service.ErrEventTrackNotFound),
		errors.Is(err, service.ErrTrackNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidEvent):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrVoteNotAllowed),
		errors.Is(err, service.ErrVoteWindowClosed),
		errors.Is(err, service.ErrOutsideVoteArea),
		errors.Is(err, service.ErrLocationRequired):
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrPlaylistNotFound), errors.Is(err, service.ErrPlaylistTrackNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrVersionConflict), errors.Is(err, service.ErrTrackAlreadyInPlaylist):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidMove):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
}
