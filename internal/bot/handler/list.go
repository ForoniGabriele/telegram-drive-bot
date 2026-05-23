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
	cat := Cat(c)
	lang := Lang(c)
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return nil
	}
	fileID := data.FileDBID

	// 必须用 user.ID 限定:callback_data 来自客户端,可被构造任意 fileID 越权访问他人文件
	file, err := h.fileService.GetFileByIDAndUser(fileID, user.ID)
	if err != nil || file == nil {
		return c.RespondText(cat.FileNotFound)
	}

	caption := buildFileCaption(cat, lang, file)

	copies, err := h.fileService.GetFileCopies(file)
	if err != nil {
		slog.Error("load file copies failed", "error", err, "file_db_id", file.ID)
	}

	sent, err := storage.SendWithFallback(c.Bot(), h.storage, c.Chat(), file, copies, caption)
	if err == nil {
		attachDeleteButton(c.Bot(), cat, sent, file.ID)
		return c.Respond()
	}
	if errors.Is(err, storage.ErrFileUnsendable) {
		return c.Send(cat.FileUnavailable)
	}
	slog.Error("send file to user failed", "error", err, "file_db_id", file.ID)
	return c.Send(cat.FileSendFailed)
}

// sendListPage sends or edits a file list message with pagination and type filter.
func (h *ListHandler) sendListPage(c tele.Context, userID uint, fileType string, page int, isEdit bool) error {
	cat := Cat(c)
	lang := Lang(c)
	files, total, err := h.fileService.GetFilesByUser(userID, fileType, page, util.DefaultPageSize)
	if err != nil {
		slog.Error("list files failed", "error", err, "user_id", userID, "file_type", fileType, "page", page)
		return c.Send(cat.ListFetchFailed)
	}

	pag := &util.Pagination{Page: page, PageSize: util.DefaultPageSize, Total: total}

	text := formatListText(cat, lang, files, total, fileType, page, h.botUsername)
	markup := buildListKeyboard(cat, files, fileType, pag)

	return ui.EditOrSend(c, text, markup, isEdit)
}

// formatListText composes the list title and body.
func formatListText(cat *msg.Catalog, lang string, files []model.File, total int64, fileType string, page int, botUsername string) string {
	if total == 0 {
		if fileType != constants.FileTypeAll.String() && fileType != "" {
			return fmt.Sprintf(cat.ListEmptyType, util.FileTypeIcon(fileType), util.FileTypeName(lang, fileType))
		}
		return cat.ListEmpty
	}
	header := fmt.Sprintf(cat.ListHeader, total)
	return ui.FormatFileList(lang, header, files, page, util.DefaultPageSize, botUsername)
}

// buildListKeyboard assembles the list inline keyboard: file number row(s) + type filter + pagination.
func buildListKeyboard(cat *msg.Catalog, files []model.File, fileType string, pag *util.Pagination) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	rows = append(rows, ui.FileNumberButtons(markup, files, pag.Page, pag.PageSize)...)

	rows = append(rows, ui.FileTypeFilterRow(markup, cat, ui.CBList, func(ft constants.FileType) string {
		return ui.Encode(ui.CBData{FileType: ft.String(), Page: 1})
	}))

	rows = append(rows, ui.PaginationRow(markup, cat, ui.CBList, pag, func(p int) string {
		return ui.Encode(ui.CBData{FileType: fileType, Page: p})
	}))

	markup.Inline(rows...)
	return markup
}

// OnFileDelete handles the first tap on the delete button — swap keyboard to confirm/cancel.
func (h *ListHandler) OnFileDelete(c tele.Context) error {
	cat := Cat(c)
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return c.Respond()
	}
	if _, err := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileDeleteConfirmKeyboard(cat, data.FileDBID)); err != nil {
		slog.Warn("swap to confirm keyboard failed", "error", err, "file_db_id", data.FileDBID)
	}
	return c.Respond()
}

// OnFileDeleteCancel handles the cancel button — restore the original single-button keyboard.
func (h *ListHandler) OnFileDeleteCancel(c tele.Context) error {
	cat := Cat(c)
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.FileDBID == 0 {
		return c.Respond()
	}
	if _, err := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileActionKeyboard(cat, data.FileDBID)); err != nil {
		slog.Warn("restore delete keyboard failed", "error", err, "file_db_id", data.FileDBID)
	}
	return c.Respond()
}

// OnFileDeleteConfirm handles the confirm button — performs the actual deletion through the service layer.
func (h *ListHandler) OnFileDeleteConfirm(c tele.Context) error {
	cat := Cat(c)
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
			if _, editErr := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileActionKeyboard(cat, fileID)); editErr != nil {
				slog.Warn("restore keyboard after not-found failed", "error", editErr, "file_db_id", fileID)
			}
			return c.Respond(&tele.CallbackResponse{Text: cat.FileNotFoundOrNoPerm, ShowAlert: true})
		}
		slog.Error("delete file failed", "error", err, "file_db_id", fileID, "user_id", user.ID)
		if _, editErr := c.Bot().EditReplyMarkup(c.Callback().Message, ui.FileActionKeyboard(cat, fileID)); editErr != nil {
			slog.Warn("restore keyboard after failure failed", "error", editErr, "file_db_id", fileID)
		}
		return c.Respond(&tele.CallbackResponse{Text: cat.FileDeleteFailed, ShowAlert: true})
	}

	// Clear the keyboard so the message stays but no longer offers the delete action.
	if _, err := c.Bot().EditReplyMarkup(c.Callback().Message, &tele.ReplyMarkup{}); err != nil {
		slog.Warn("clear keyboard after delete failed", "error", err, "file_db_id", fileID)
	}
	return c.Respond(&tele.CallbackResponse{Text: cat.FileDeleted})
}

func buildFileCaption(cat *msg.Catalog, lang string, file *model.File) string {
	icon := util.FileTypeIcon(file.FileType)
	size := util.FormatFileSize(file.FileSize)
	typeName := util.FileTypeName(lang, file.FileType)

	fileName := file.FileName
	if fileName == "" {
		fileName = typeName
	}

	header := fmt.Sprintf(cat.FileCaptionHeader, icon, fileName, size, file.CreatedAt.Format(constants.DateFull))

	if file.Caption == "" {
		return util.TruncateCaption(header, constants.CaptionMaxUTF16)
	}

	// caption 上限按 UTF-16 计 -- 优先保留 header, 不够时再截断用户 caption
	headerLen := util.CaptionUTF16Len(header)
	const sepLen = 2 // "\n\n"
	remaining := constants.CaptionMaxUTF16 - headerLen - sepLen
	if remaining <= 0 {
		return util.TruncateCaption(header, constants.CaptionMaxUTF16)
	}
	return header + "\n\n" + util.TruncateCaption(file.Caption, remaining)
}
