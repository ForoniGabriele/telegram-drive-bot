package bot

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"tg-drive-bot/internal/bot/handler"
	"tg-drive-bot/internal/bot/middleware"
	"tg-drive-bot/internal/bot/ui"
	"tg-drive-bot/internal/config"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/storage"
	"tg-drive-bot/internal/telegraph"
	"tg-drive-bot/internal/util"

	"golang.org/x/net/proxy"
	tele "gopkg.in/telebot.v4"
)

// newProxyClient creates an http.Client with the given proxy URL.
// Supports socks5:// and http:// (also https://) proxy schemes.
func newProxyClient(proxyURL string) (*http.Client, error) {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	switch parsed.Scheme {
	case "socks5":
		var auth *proxy.Auth
		if parsed.User != nil {
			password, _ := parsed.User.Password()
			auth = &proxy.Auth{
				User:     parsed.User.Username(),
				Password: password,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}
		return &http.Client{
			Timeout: time.Minute,
			Transport: &http.Transport{
				Dial: dialer.Dial,
			},
		}, nil

	case "http", "https":
		return &http.Client{
			Timeout: time.Minute,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsed),
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s (use socks5:// or http://)", parsed.Scheme)
	}
}

// New creates and configures a new telebot Bot instance with all handlers and middleware.
func New(cfg *config.Config, userService *service.UserService, fileService *service.FileService, embeddingSvc *service.EmbeddingService, maintenanceSvc *service.MaintenanceService, store storage.Storage) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  cfg.Bot.Token,
		Poller: &tele.LongPoller{Timeout: 40 * time.Second},
		OnError: func(err error, c tele.Context) {
			if c != nil && c.Sender() != nil {
				slog.Error("handler error", "error", err, "user", c.Sender().ID, "chat", c.Chat().ID)
			} else {
				slog.Error("bot error", "error", err)
			}
		},
	}

	// Configure proxy if specified
	var httpClient *http.Client
	if cfg.Bot.ProxyURL != "" {
		client, err := newProxyClient(cfg.Bot.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("failed to configure proxy: %w", err)
		}
		httpClient = client
		pref.Client = client
		slog.Info("using proxy", "url", cfg.Bot.ProxyURL)
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	// Initialize infrastructure
	botUsername := b.Me.Username
	searchCache := util.NewSearchCache()
	tphClient := telegraph.NewClient(httpClient)

	// Initialize handlers
	mediaHandler := handler.NewMediaHandler(fileService, store)
	listHandler := handler.NewListHandler(fileService, store, botUsername)
	searchHandler := handler.NewSearchHandler(fileService, store, searchCache, botUsername)
	startHandler := handler.NewStartHandler(fileService, store)
	statsHandler := handler.NewStatsHandler(fileService)
	randomHandler := handler.NewRandomHandler(fileService, store)
	adminHandler := handler.NewAdminUserHandler(userService)
	telegraphHandler := handler.NewTelegraphHandler(tphClient, fileService, store)
	embAdminHandler := handler.NewEmbeddingAdminHandler(embeddingSvc)
	maintAdminHandler := handler.NewMaintenanceAdminHandler(maintenanceSvc)

	// ========== Global Middleware ==========
	b.Use(middleware.Whitelist(userService))

	// ========== User Commands ==========
	b.Handle("/start", startHandler.OnStart)
	b.Handle("/list", listHandler.OnList)
	b.Handle("/search", searchHandler.OnSearch)
	b.Handle("/ss", searchHandler.OnQuickSearch)
	b.Handle("/stats", statsHandler.OnStats)
	b.Handle("/rand", randomHandler.OnRand)
	b.Handle("/randv", randomHandler.OnRandVideo)
	b.Handle("/randp", randomHandler.OnRandPhoto)
	b.Handle("/tph", telegraphHandler.OnTelegraph)

	// ========== Media Events ==========
	mediaEvents := []string{
		tele.OnDocument,
		tele.OnPhoto,
		tele.OnVideo,
		tele.OnAudio,
		tele.OnAnimation,
		tele.OnVoice,
		tele.OnVideoNote,
	}
	for _, event := range mediaEvents {
		b.Handle(event, mediaHandler.OnMediaReceived)
	}

	// ========== Callback Handlers ==========
	// File retrieval (shared by list and search)
	btnFile := tele.InlineButton{Unique: ui.CBFile.String()}
	b.Handle(&btnFile, listHandler.OnFileCallback)

	// List pagination and type filter
	btnList := tele.InlineButton{Unique: ui.CBList.String()}
	b.Handle(&btnList, listHandler.OnListCallback)

	// Search pagination and type filter
	btnSearch := tele.InlineButton{Unique: ui.CBSearch.String()}
	b.Handle(&btnSearch, searchHandler.OnSearchCallback)

	// No-op button (page indicator)
	btnNoop := tele.InlineButton{Unique: ui.CBNoop.String()}
	b.Handle(&btnNoop, func(c tele.Context) error {
		return c.Respond()
	})

	// File deletion flow: request → confirm/cancel
	btnFileDel := tele.InlineButton{Unique: ui.CBFileDel.String()}
	b.Handle(&btnFileDel, listHandler.OnFileDelete)

	btnFileDelConfirm := tele.InlineButton{Unique: ui.CBFileDelConfirm.String()}
	b.Handle(&btnFileDelConfirm, listHandler.OnFileDeleteConfirm)

	btnFileDelCancel := tele.InlineButton{Unique: ui.CBFileDelCancel.String()}
	b.Handle(&btnFileDelCancel, listHandler.OnFileDeleteCancel)

	// ========== Admin Commands (Group with AdminOnly middleware) ==========
	admins := b.Group()
	admins.Use(middleware.AdminOnly())
	admins.Handle("/adduser", adminHandler.OnAddUser)
	admins.Handle("/removeuser", adminHandler.OnRemoveUser)
	admins.Handle("/listuser", adminHandler.OnListUser)

	// Admin callback handlers
	btnUserList := tele.InlineButton{Unique: ui.CBUserList.String()}
	b.Handle(&btnUserList, adminHandler.OnUserListCallback)

	btnUserInfo := tele.InlineButton{Unique: ui.CBUserInfo.String()}
	b.Handle(&btnUserInfo, adminHandler.OnUserInfoCallback)

	btnUserDel := tele.InlineButton{Unique: ui.CBUserDel.String()}
	b.Handle(&btnUserDel, adminHandler.OnUserDeleteCallback)

	btnUserPromote := tele.InlineButton{Unique: ui.CBUserPromote.String()}
	b.Handle(&btnUserPromote, adminHandler.OnUserPromoteCallback)

	btnUserDemote := tele.InlineButton{Unique: ui.CBUserDemote.String()}
	b.Handle(&btnUserDemote, adminHandler.OnUserDemoteCallback)

	// ========== Owner Commands (Group with OwnerOnly middleware) ==========
	// 这些命令通常是耗时 / 影响范围广的运维操作:批量 embedding、caption 同步等
	owners := b.Group()
	owners.Use(middleware.OwnerOnly())
	owners.Handle("/emb_re", embAdminHandler.OnEmbRe)
	owners.Handle("/emb_sync", embAdminHandler.OnEmbSync)
	owners.Handle("/cap_sync", maintAdminHandler.OnCapSync)

	return b, nil
}
