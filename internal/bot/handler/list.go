package handler

import (
	"errors"
	"fmt"
	"log/slog"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/bot/ui"
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/storage"
	"tg-drive-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

// ListHandler handles the /list command and its callbacks.
type ListHandler struct {
	fileService *service.FileService
	storage     storage.Storage
	botUsername string
}

// NewListHandler creates a new ListHandler.
func NewListHandler(fileService *service.FileService, store storage.Storage, botUsername string) *ListHandler {
	return &ListHandler{
		fileService: fileService,
		storage:     store,
		botUsername: botUsername,
	}
}

// OnList handles the /list command.
func (h *ListHandler) OnList(c tele.Context) error {
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}
	return h.sendListPage(c, user.ID, constants.FileTypeAll.String(), 1, false)
}

// OnListCallback handles list pagination and type filter callbacks.
func (h *ListHandler) OnListCallback(c tele.Context) error {
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}

	data, err := ui.Decode(c.Callback().Data)
	if err != nil {
		return nil
	}

	fileType := data.FileType
	if fileType == "" {
		fileType = constants.FileTypeAll.String()
	}
	page := data.Page
	if page < 1 {
		page = 1
	}

	return h.sendListPage(c, user.ID, fileType, page, true)
}

// OnFileCallback handles file retrieval callbacks.
func (h *ListHandler) OnFileCallback(c tele.Context) error {
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return nil
	}
	fileID := data.FileDBID

	file, err := h.fileService.GetFileByID(fileID)
	if err != nil || file == nil {
		return c.RespondText(msg.FileNotFound)
	}

	caption := buildFileCaption(file)

	copies, err := h.fileService.GetFileCopies(file)
	if err != nil {
		slog.Error("load file copies failed", "error", err, "file_db_id", file.ID)
	}

	sent, err := h.storage.SendFileToUser(c.Bot(), c.Chat(), file, copies, caption)
	if err == nil {
		attachDeleteButton(c.Bot(), sent, file.ID)
		return c.Respond()
	}

	// All channel copies failed — fall back to a direct file_id send so the user still gets the file.
	if errors.Is(err, storage.ErrAllCopiesFailed) {
		slog.Warn("all storage copies failed, falling back to direct send", "file_db_id", file.ID)
		if sendable := storage.BuildSendable(file, caption); sendable != nil {
			if fallback, sendErr := c.Bot().Send(c.Chat(), sendable); sendErr == nil {
				attachDeleteButton(c.Bot(), fallback, file.ID)
				return c.Respond()
			} else {
				slog.Error("direct-send fallback failed", "error", sendErr, "file_db_id", file.ID)
			}
		}
		return c.Send(msg.FileUnavailable)
	}

	slog.Error("send file to user failed", "error", err, "file_db_id", file.ID)
	return c.Send(msg.FileSendFailed)
}

// sendListPage sends or edits a file list message with pagination and type filter.
func (h *ListHandler) sendListPage(c tele.Context, userID uint, fileType string, page int, isEdit bool) error {
	files, total, err := h.fileService.GetFilesByUser(userID, fileType, page, util.DefaultPageSize)
	if err != nil {
		slog.Error("list files failed", "error", err, "user_id", userID, "file_type", fileType, "page", page)
		return c.Send(msg.ListFetchFailed)
	}

	pag := &util.Pagination{Page: page, PageSize: util.DefaultPageSize, Total: total}

	text := formatListText(files, total, fileType, page, h.botUsername)
	markup := buildListKeyboard(files, fileType, pag)

	return ui.EditOrSend(c, text, markup, isEdit)
}

// formatListText composes the list title and body.
func formatListText(files []model.File, total int64, fileType string, page int, botUsername string) string {
	if total == 0 {
		if fileType != constants.FileTypeAll.String() && fileType != "" {
			return fmt.Sprintf("📁 我的文件库\n\n没有 %s %s 类型的文件",
				util.FileTypeIcon(fileType), util.FileTypeName(fileType))
		}
		return "📁 我的文件库\n\n还没有存储任何文件, 直接发送文件给我即可开始!"
	}
	header := fmt.Sprintf("📁 我的文件库 (共 %d 个文件)\n\n", total)
	return ui.FormatFileList(header, files, page, util.DefaultPageSize, botUsername)
}

// buildListKeyboard assembles the list inline keyboard: file number row(s) + type filter + pagination.
func buildListKeyboard(files []model.File, fileType string, pag *util.Pagination) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	rows = append(rows, ui.FileNumberButtons(markup, files, pag.Page, pag.PageSize)...)

	rows = append(rows, ui.FileTypeFilterRow(markup, ui.CBList, func(ft constants.FileType) string {
		return ui.Encode(ui.CBData{FileType: ft.String(), Page: 1})
	}))

	rows = append(rows, ui.PaginationRow(markup, ui.CBList, pag, func(p int) string {
		return ui.Encode(ui.CBData{FileType: fileType, Page: p})
	}))

	markup.Inline(rows...)
	return markup
}

// OnFileDelete handles the first tap on "🗑 删除" — swap keyboard to confirm/cancel.
func (h *ListHandler) OnFileDelete(c tele.Context) error {
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return c.Respond()
	}
	if _, err := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileDeleteConfirmKeyboard(data.FileDBID)); err != nil {
		slog.Warn("swap to confirm keyboard failed", "error", err, "file_db_id", data.FileDBID)
	}
	return c.Respond()
}

// OnFileDeleteCancel handles "❌ 取消" — restore the original single-button keyboard.
func (h *ListHandler) OnFileDeleteCancel(c tele.Context) error {
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return c.Respond()
	}
	if _, err := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileActionKeyboard(data.FileDBID)); err != nil {
		slog.Warn("restore delete keyboard failed", "error", err, "file_db_id", data.FileDBID)
	}
	return c.Respond()
}

// OnFileDeleteConfirm handles "✅ 确认删除" — performs the actual deletion through the service layer.
func (h *ListHandler) OnFileDeleteConfirm(c tele.Context) error {
	user, ok := RequireUser(c)
	if !ok {
		return c.Respond()
	}
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return c.Respond()
	}
	fileID := data.FileDBID

	err = h.fileService.DeleteFile(user.ID, fileID, c.Bot(), h.storage)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			if _, editErr := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileActionKeyboard(fileID)); editErr != nil {
				slog.Warn("restore keyboard after not-found failed", "error", editErr, "file_db_id", fileID)
			}
			return c.Respond(&tele.CallbackResponse{Text: msg.FileNotFoundOrNoPerm, ShowAlert: true})
		}
		slog.Error("delete file failed", "error", err, "file_db_id", fileID, "user_id", user.ID)
		if _, editErr := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileActionKeyboard(fileID)); editErr != nil {
			slog.Warn("restore keyboard after failure failed", "error", editErr, "file_db_id", fileID)
		}
		return c.Respond(&tele.CallbackResponse{Text: msg.FileDeleteFailed, ShowAlert: true})
	}

	// Clear the keyboard so the message stays but no longer offers the delete action.
	if _, err := c.Bot().EditReplyMarkup(c.Callback().Message, &tele.ReplyMarkup{}); err != nil {
		slog.Warn("clear keyboard after delete failed", "error", err, "file_db_id", fileID)
	}
	return c.Respond(&tele.CallbackResponse{Text: msg.FileDeleted})
}

func buildFileCaption(file *model.File) string {
	icon := util.FileTypeIcon(file.FileType)
	size := util.FormatFileSize(file.FileSize)
	typeName := util.FileTypeName(file.FileType)

	fileName := file.FileName
	if fileName == "" {
		fileName = typeName
	}

	text := fmt.Sprintf(
		"📋 文件信息\n━━━━━━━━━━━━━━━\n%s 文件名: %s\n📦 大小: %s\n📅 存入时间: %s\n━━━━━━━━━━━━━━━",
		icon, fileName, size, file.CreatedAt.Format(constants.DateFull),
	)

	if file.Caption != "" {
		text += fmt.Sprintf("\n\n%s", file.Caption)
	}

	return text
}
