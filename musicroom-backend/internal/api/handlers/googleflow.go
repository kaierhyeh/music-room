package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/auth"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type GoogleFlowHandler struct {
	svc     *service.AuthService
	oauth   *auth.GoogleOAuth
	states  *returnStore
	handoff *handoffStore
	link    *auth.TicketStore
	appURL  string
	scheme  string
}

func NewGoogleFlowHandler(svc *service.AuthService, oauth *auth.GoogleOAuth,
	appURL string) *GoogleFlowHandler {
	scheme := ""
	if u, err := url.Parse(appURL); err == nil {
		scheme = strings.ToLower(u.Scheme)
	}
	return &GoogleFlowHandler{
		svc:     svc,
		oauth:   oauth,
		states:  newReturnStore(15 * time.Minute),
		link:    auth.NewTicketStore(2 * time.Minute),
		handoff: newHandoffStore(2 * time.Minute),
		appURL:  appURL,
		scheme:  scheme,
	}
}

// @Summary     Begin Google sign-in (redirects the browser to Google)
// @Tags        auth
// @Param       redirect query string false "Deep link to return to; defaults to APP_REDIRECT_URL"
// @Param       link     query string false "Link ticket from /users/me/oauth/google/link-ticket; turns the flow into a link instead of a sign-in"
// @Param       code_challenge query string false "PKCE S256 challenge; /exchange then demands its verifier"
// @Success     302
// @Failure     503 {object} map[string]string
// @Router      /auth/google/start [get]
func (h *GoogleFlowHandler) Start(c echo.Context) error {
	if !h.oauth.Configured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "google sign-in is not configured")
	}
	challenge := c.QueryParam("code_challenge")
	if challenge != "" && !validChallenge(challenge) {
		return echo.NewHTTPError(http.StatusBadRequest, "code_challenge must be an S256 challenge")
	}
	if m := c.QueryParam("code_challenge_method"); m != "" && m != "S256" {
		return echo.NewHTTPError(http.StatusBadRequest, "only S256 is supported")
	}
	linkTo, _, _ := h.link.Consume(c.QueryParam("link"))
	back := h.resolveReturn(c.QueryParam("redirect"), challenge != "")
	state, err := h.states.Issue(back, linkTo, challenge)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.Redirect(http.StatusFound, h.oauth.AuthCodeURL(state))
}

// @Summary     Issue a single-use ticket authorizing a Google link for the caller
// @Tags        auth
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} map[string]string
// @Router      /users/me/oauth/google/link-ticket [post]
func (h *GoogleFlowHandler) LinkTicket(c echo.Context) error {
	if !h.oauth.Configured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "google sign-in is not configured")
	}
	tok, err := h.link.Issue(mw.UserID(c), mw.Client(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
	}
	return c.JSON(http.StatusOK, map[string]string{"ticket": tok})
}

// @Summary     Google redirects here; bounces back to the app deep link
// @Tags        auth
// @Param       code  query string false "Authorization code"
// @Param       state query string false "Opaque state issued by /auth/google/start"
// @Success     302
// @Router      /auth/google/callback [get]
func (h *GoogleFlowHandler) Callback(c echo.Context) error {
	if !h.oauth.Configured() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "google sign-in is not configured")
	}
	back, linkTo, challenge, ok := h.states.Consume(c.QueryParam("state"))
	if !ok {
		return c.HTML(http.StatusBadRequest, expiredPage)
	}
	if e := c.QueryParam("error"); e != "" {
		return h.backToApp(c, back, "error", e)
	}
	code := c.QueryParam("code")
	if code == "" {
		return h.backToApp(c, back, "error", "missing_code")
	}

	ctx := c.Request().Context()
	idToken, err := h.oauth.ExchangeCode(ctx, code)
	if err != nil {
		log.Printf("google flow: %v", err)
		return h.backToApp(c, back, "error", "exchange_failed")
	}
	if linkTo != uuid.Nil {
		if err := h.svc.LinkGoogle(ctx, linkTo, idToken); err != nil {
			log.Printf("google flow: link: %v", err)
			if errors.Is(err, service.ErrOAuthConflict) {
				return h.backToApp(c, back, "error", "already_linked_elsewhere")
			}
			return h.backToApp(c, back, "error", "link_failed")
		}
		return h.backToApp(c, back, "linked", "1")
	}

	_, user, err := h.svc.OAuthGoogle(ctx, idToken, reqMeta(c))
	if err != nil {
		log.Printf("google flow: sign-in: %v", err)
		return h.backToApp(c, back, "error", "sign_in_failed")
	}

	handoff, err := h.handoff.Issue(user.ID, challenge)
	if err != nil {
		return h.backToApp(c, back, "error", "internal")
	}
	return h.backToApp(c, back, "code", handoff)
}

// @Summary     Trade the one-time code from the deep link for a session
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body googleExchangeRequest true "One-time code"
// @Success     200 {object} tokenResponse
// @Failure     400 {object} map[string]string
// @Router      /auth/google/exchange [post]
func (h *GoogleFlowHandler) Exchange(c echo.Context) error {
	var req googleExchangeRequest
	if err := bindValidate(c, &req); err != nil {
		return err
	}
	userID, challenge, ok := h.handoff.Consume(req.Code)
	if !ok || userID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired code")
	}
	if challenge != "" && !verifierMatches(req.CodeVerifier, challenge) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired code")
	}
	pair, err := h.svc.IssueTokensFor(c.Request().Context(), userID, reqMeta(c))
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(http.StatusOK, toTokenResponse(pair))
}

type googleExchangeRequest struct {
	Code         string `json:"code" validate:"required"`
	CodeVerifier string `json:"code_verifier,omitempty"`
}

func validChallenge(ch string) bool {
	b, err := base64.RawURLEncoding.DecodeString(ch)
	return err == nil && len(b) == sha256.Size
}

func verifierMatches(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(got), []byte(challenge)) == 1
}

func (h *GoogleFlowHandler) backToApp(c echo.Context, appURL, key, value string) error {
	sep := "?"
	if strings.Contains(appURL, "?") {
		sep = "&"
	}
	return c.Redirect(http.StatusFound, appURL+sep+key+"="+url.QueryEscape(value))
}

const expiredPage = `<!doctype html><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sign-in expired</title>
<style>
 body{margin:0;min-height:100vh;display:grid;place-items:center;
      background:#0b0b0b;color:#c8c8c8;
      font:16px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace}
 main{max-width:30rem;padding:2rem;text-align:center}
 h1{color:#4ade80;font-size:1.25rem;letter-spacing:.08em;text-transform:uppercase}
</style>
<main>
 <h1>Sign-in expired</h1>
 <p>This sign-in took too long, or was already completed.</p>
 <p>Close this tab and tap <strong>Sign in with Google</strong> in MusicRoom again.</p>
</main>`

func (h *GoogleFlowHandler) resolveReturn(raw string, pkce bool) string {
	if raw == "" {
		return h.appURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return h.appURL
	}
	switch strings.ToLower(u.Scheme) {
	case "":
		return h.appURL
	case "exp":
		if pkce && u.Host != "" {
			return raw
		}
		if isLocalHost(u.Hostname()) {
			return raw
		}
	case "http", "https":
		if isLocalHost(u.Hostname()) {
			return raw
		}
	case h.scheme:
		return raw
	}
	return h.appURL
}

func isLocalHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

type returnStore struct {
	mu  sync.Mutex
	m   map[string]returnEntry
	ttl time.Duration
}

type returnEntry struct {
	url       string
	linkTo    uuid.UUID
	challenge string
	expires   time.Time
}

func newReturnStore(ttl time.Duration) *returnStore {
	return &returnStore{m: make(map[string]returnEntry), ttl: ttl}
}

func (s *returnStore) Issue(appURL string, linkTo uuid.UUID, challenge string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, e := range s.m {
		if now.After(e.expires) {
			delete(s.m, k)
		}
	}
	s.m[tok] = returnEntry{url: appURL, linkTo: linkTo, challenge: challenge, expires: now.Add(s.ttl)}
	return tok, nil
}

func (s *returnStore) Consume(tok string) (string, uuid.UUID, string, bool) {
	if tok == "" {
		return "", uuid.Nil, "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[tok]
	if !ok {
		return "", uuid.Nil, "", false
	}
	delete(s.m, tok)
	if time.Now().After(e.expires) {
		return "", uuid.Nil, "", false
	}
	return e.url, e.linkTo, e.challenge, true
}

type handoffStore struct {
	mu  sync.Mutex
	m   map[string]handoffEntry
	ttl time.Duration
}

type handoffEntry struct {
	userID    uuid.UUID
	challenge string
	expires   time.Time
}

func newHandoffStore(ttl time.Duration) *handoffStore {
	return &handoffStore{m: make(map[string]handoffEntry), ttl: ttl}
}

func (s *handoffStore) Issue(userID uuid.UUID, challenge string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, e := range s.m {
		if now.After(e.expires) {
			delete(s.m, k)
		}
	}
	s.m[tok] = handoffEntry{userID: userID, challenge: challenge, expires: now.Add(s.ttl)}
	return tok, nil
}

func (s *handoffStore) Consume(tok string) (uuid.UUID, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[tok]
	if !ok {
		return uuid.Nil, "", false
	}
	delete(s.m, tok)
	if time.Now().After(e.expires) {
		return uuid.Nil, "", false
	}
	return e.userID, e.challenge, true
}
