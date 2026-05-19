package handler

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/storage"
	"tg-drive-bot/internal/telegraph"

	tele "gopkg.in/telebot.v4"
)

const (
	tphMarkdownMIME = "text/markdown; charset=utf-8"
	tphFileNameMax  = 120
)

// TelegraphHandler handles the /tph command: fetches a Telegra.ph article,
// renders it to Markdown, uploads the file to Telegram, and indexes it into
// the user's library using the same storage pipeline as regular media uploads.
type TelegraphHandler struct {
	client      *telegraph.Client
	fileService *service.FileService
	storage     storage.Storage
}

// NewTelegraphHandler creates a new TelegraphHandler.
func NewTelegraphHandler(client *telegraph.Client, fileService *service.FileService, store storage.Storage) *TelegraphHandler {
	return &TelegraphHandler{
		client:      client,
		fileService: fileService,
		storage:     store,
	}
}

// OnTelegraph handles the /tph command.
func (h *TelegraphHandler) OnTelegraph(c tele.Context) error {
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}

	arg := strings.TrimSpace(c.Message().Payload)
	if arg == "" {
		return c.Send(msg.TphUsage)
	}

	path, err := telegraph.ExtractPath(arg)
	if err != nil {
		slog.Info("telegraph invalid url", "input", arg, "error", err)
		return c.Send(msg.TphInvalidURL)
	}

	// Acknowledge so the user sees progress on slow networks.
	_ = c.Send(msg.TphFetching)

	page, err := h.client.GetPage(context.Background(), path)
	if err != nil {
		slog.Warn("telegraph fetch failed", "path", path, "error", err)
		return c.Send(msg.TphFetchFailed)
	}
	if len(page.Content) == 0 {
		return c.Send(msg.TphEmptyContent)
	}

	markdown := telegraph.RenderMarkdown(page)
	content := []byte(markdown)
	fileName := buildTphFileName(page)

	doc := &tele.Document{
		File:     tele.FromReader(bytes.NewReader(content)),
		FileName: fileName,
		MIME:     tphMarkdownMIME,
	}

	sent, err := c.Bot().Send(c.Chat(), doc)
	if err != nil {
		slog.Error("telegraph upload failed", "error", err, "user_id", user.ID, "path", path)
		return c.Send(msg.TphFetchFailed)
	}
	if sent.Document == nil {
		slog.Error("telegraph upload returned no document", "user_id", user.ID, "path", path)
		return c.Send(msg.TphFetchFailed)
	}

	// Replicate the regular media pipeline: check duplicate, forward to storage, save.
	fileUniqueID := sent.Document.UniqueID
	if fileUniqueID == "" {
		// Telegram should always return a UniqueID; bail rather than insert a row
		// that would collide with itself on re-upload.
		slog.Error("telegraph upload missing file_unique_id", "user_id", user.ID, "path", path)
		return c.Send(msg.TphFetchFailed)
	}

	isDup, err := h.fileService.CheckDuplicate(user.ID, fileUniqueID)
	if err != nil {
		slog.Error("telegraph duplicate check failed", "error", err, "user_id", user.ID, "file_unique_id", fileUniqueID)
		return c.Send(msg.FileSaveFailed)
	}
	if isDup {
		// Same article previously saved in this chat — silently skip indexing,
		// but the fresh upload is already visible in the chat.
		return nil
	}

	info := &service.FileInfo{
		FileType:     constants.FileTypeDocument.String(),
		BotFileID:    sent.Document.FileID,
		FileUniqueID: fileUniqueID,
		FileName:     fileName,
		MimeType:     tphMarkdownMIME,
		FileSize:     int64(len(content)),
		Caption:      page.Title,
	}
	msgInfo := &service.MessageInfo{
		MessageID:    sent.ID,
		ChatID:       sent.Chat.ID,
		FromUserID:   c.Sender().ID,
		FromUsername: c.Sender().Username,
		Caption:      page.Title,
		ForwardFrom:  fmt.Sprintf("telegraph:%s", path),
	}

	if _, err := h.fileService.SaveFile(user.ID, info, msgInfo, c.Bot(), sent, h.storage); err != nil {
		slog.Error("telegraph save file failed", "error", err, "user_id", user.ID, "path", path)
		return c.Send(msg.FileSaveFailed)
	}

	return c.Send(fmt.Sprintf(msg.TphSaved, fileName))
}

// tphFileNameSanitizer matches characters Telegram/Windows/Linux commonly reject
// in file names. Collapses them to '_'.
var tphFileNameSanitizer = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// buildTphFileName derives a safe .md filename from the page path (fallback: title).
func buildTphFileName(page *telegraph.Page) string {
	base := strings.TrimSpace(page.Path)
	if base == "" {
		base = strings.TrimSpace(page.Title)
	}
	if base == "" {
		base = "telegraph"
	}
	base = tphFileNameSanitizer.ReplaceAllString(base, "_")
	base = strings.Trim(base, ". ")
	runes := []rune(base)
	if len(runes) > tphFileNameMax {
		runes = runes[:tphFileNameMax]
	}
	return string(runes) + ".md"
}
