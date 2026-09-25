package api

import (
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	echoswagger "github.com/swaggo/echo-swagger"

	_ "github.com/nfauconn/music-room/musicroom-backend/docs"
	"github.com/nfauconn/music-room/musicroom-backend/internal/api/handlers"
	mw "github.com/nfauconn/music-room/musicroom-backend/internal/api/middleware"
	"github.com/nfauconn/music-room/musicroom-backend/internal/audit"
	"github.com/nfauconn/music-room/musicroom-backend/internal/auth"
	"github.com/nfauconn/music-room/musicroom-backend/internal/config"
	"github.com/nfauconn/music-room/musicroom-backend/internal/deezer"
	"github.com/nfauconn/music-room/musicroom-backend/internal/hub"
	"github.com/nfauconn/music-room/musicroom-backend/internal/mail"
	"github.com/nfauconn/music-room/musicroom-backend/internal/repository"
	"github.com/nfauconn/music-room/musicroom-backend/internal/service"
)

type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i any) error { return cv.v.Struct(i) }

func NewRouter(cfg *config.Config, pool *pgxpool.Pool) (*echo.Echo, func()) {
	e := echo.New()
	e.HideBanner = true
	e.Validator = &customValidator{v: validator.New()}

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.BodyLimit("1M"))
	rpsLimit := cfg.RateLimitRPS
	if rpsLimit <= 0 {
		rpsLimit = 20
	}
	authPerMin := cfg.RateLimitAuthPerMinute
	if authPerMin <= 0 {
		authPerMin = 10
	}
	authBurst := cfg.RateLimitAuthBurst
	if authBurst <= 0 {
		authBurst = 3
	}
	if cfg.RateLimitDisabled {
		log.Printf("WARNING: rate limiting is disabled (RATE_LIMIT_DISABLED=true). " +
			"Intended for load testing from a single host; never run a reachable server this way.")
	} else {
		e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(rpsLimit))))
	}
	if len(cfg.CORSOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: cfg.CORSOrigins}))
	}

	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTAccessTTL)
	mailer := mail.New(mail.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	googleVerifier := auth.NewGoogleVerifier(cfg.GoogleClientID)
	googleOAuth := auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	authSvc := service.NewAuthService(pool, jwtMgr, mailer, googleVerifier, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	q := repository.New(pool)

	userSvc := service.NewUserService(q)
	friendSvc := service.NewFriendService(pool)

	wsHub := hub.New()
	dz := deezer.New(cfg.DeezerAPIBase)
	eventSvc := service.NewEventService(pool, dz, wsHub)
	trackSvc := service.NewTrackService(dz, q)
	playlistSvc := service.NewPlaylistService(pool, dz, wsHub)
	ticketStore := auth.NewTicketStore(30 * time.Second)

	subSvc := service.NewSubscriptionService(pool)

	authH := handlers.NewAuthHandler(authSvc)
	subH := handlers.NewSubscriptionHandler(subSvc)
	googleH := handlers.NewGoogleFlowHandler(authSvc, googleOAuth, cfg.AppRedirectURL)
	userH := handlers.NewUserHandler(userSvc)
	friendH := handlers.NewFriendHandler(friendSvc)
	eventH := handlers.NewEventHandler(eventSvc)
	trackH := handlers.NewTrackHandler(trackSvc)
	playlistH := handlers.NewPlaylistHandler(playlistSvc)
	wsH := handlers.NewWSHandler(wsHub, ticketStore, cfg.CORSOrigins,
		eventSvc.CanAccessEventRoom, playlistSvc.CanAccessPlaylistRoom)

	auditRec := audit.NewRecorder(q, 1024)
	e.Use(mw.Audit(auditRec))

	e.GET("/health", healthHandler(pool))
	e.GET("/", handlers.HomePage)
	e.GET("/privacy", handlers.PrivacyPage)
	e.GET("/swagger/*", echoswagger.WrapHandler)

	e.GET("/ws/events/:id", wsH.ConnectEvent)
	e.GET("/ws/playlists/:id", wsH.ConnectPlaylist)

	ag := e.Group("/auth")
	if !cfg.RateLimitDisabled {
		ag.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:  rate.Limit(authPerMin / 60.0),
				Burst: authBurst,
			},
		)))
	}
	ag.POST("/register", authH.Register)
	ag.POST("/login", authH.Login)
	ag.POST("/refresh", authH.Refresh)
	ag.POST("/verify-email", authH.VerifyEmail)
	ag.POST("/forgot-password", authH.ForgotPassword)
	ag.POST("/reset-password", authH.ResetPassword)
	ag.POST("/google", authH.AuthGoogle)

	gg := e.Group("/auth/google")
	if !cfg.RateLimitDisabled {
		gg.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:  rate.Limit(authPerMin * 3 / 60.0),
				Burst: authBurst * 3,
			},
		)))
	}
	gg.GET("/start", googleH.Start)
	gg.GET("/callback", googleH.Callback)
	gg.POST("/exchange", googleH.Exchange)

	protected := e.Group("", mw.JWT(jwtMgr, authSvc))
	protected.POST("/auth/logout", authH.Logout)
	protected.GET("/users/me", userH.Me)
	protected.PATCH("/users/me", userH.UpdateMe)
	protected.POST("/users/me/oauth/google", authH.LinkGoogle)
	protected.DELETE("/users/me/oauth/google", authH.UnlinkGoogle)
	protected.POST("/users/me/oauth/google/link-ticket", googleH.LinkTicket)
	protected.GET("/users/:id", userH.GetByID)

	protected.GET("/users/me/subscription", subH.Get)
	protected.PUT("/users/me/subscription", subH.Set)
	protected.GET("/users/me/subscription/history", subH.History)

	protected.GET("/friends", friendH.List)
	protected.POST("/friends/requests", friendH.Send)
	protected.GET("/friends/requests", friendH.ListIncoming)
	protected.POST("/friends/requests/:id/accept", friendH.Accept)
	protected.POST("/friends/requests/:id/reject", friendH.Reject)

	protected.GET("/tracks/search", trackH.Search)
	protected.GET("/tracks/:id/preview", trackH.Preview)

	protected.POST("/events", eventH.Create)
	protected.GET("/events", eventH.List)
	protected.GET("/events/:id", eventH.Get)
	protected.POST("/events/:id/tracks", eventH.SuggestTrack)
	protected.POST("/events/:id/tracks/:trackId/vote", eventH.Vote)
	protected.DELETE("/events/:id/tracks/:trackId/vote", eventH.Unvote)
	protected.POST("/events/:id/invitations", eventH.Invite)
	protected.GET("/events/:id/invitations", eventH.ListInvitations)

	protected.POST("/playlists", playlistH.Create)
	protected.GET("/playlists", playlistH.List)
	protected.GET("/playlists/mine", playlistH.ListMine)
	protected.GET("/playlists/shared", playlistH.ListShared)
	protected.GET("/playlists/:id", playlistH.Get)
	protected.PATCH("/playlists/:id", playlistH.Update)
	protected.POST("/playlists/:id/tracks", playlistH.AddTrack)
	protected.PATCH("/playlists/:id/tracks/:trackId", playlistH.MoveTrack)
	protected.DELETE("/playlists/:id/tracks/:trackId", playlistH.RemoveTrack)
	protected.POST("/playlists/:id/collaborators", playlistH.AddCollaborator)
	protected.GET("/playlists/:id/collaborators", playlistH.ListCollaborators)
	protected.DELETE("/playlists/:id/collaborators/:userId", playlistH.RemoveCollaborator)

	protected.POST("/ws-tickets", wsH.IssueTicket)

	return e, auditRec.Close
}

func healthHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := pool.Ping(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
