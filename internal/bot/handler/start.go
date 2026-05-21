package handler

import (
	"errors"
	"log/slog"
	"strings"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/storage"

	tele "gopkg.in/telebot.v4"
)

// StartHandler handles the /start command, including deep-link file retrieval.
type StartHandler struct {
	fileService *service.FileService
	storage     storage.Storage
}

// NewStartHandler creates a new StartHandler.
func NewStartHandler(fileService *service.FileService, store storage.Storage) *StartHandler {
	return &StartHandler{
		fileService: fileService,
		storage:     store,
	}
}

// 用于处理 /start 指令
// 当携带格式为 "file_<FileUniqueID>" 的deep-link载荷被调用时，它会获取并
// 将该文件投递给发起请求的用户, 但仅当该文件属于该用户自己时才会进行投递
// 如果未携带载荷，则回落并发送欢迎消息
func (h *StartHandler) OnStart(c tele.Context) error {
	payload := strings.TrimSpace(c.Message().Payload)

	if strings.HasPrefix(payload, "file_") {
		// Require authenticated user for file retrieval.
		user, ok := RequireUser(c)
		if !ok {
			return nil
		}
		fileUniqueID := strings.TrimPrefix(payload, "file_")
		if fileUniqueID == "" {
			return c.Send(msg.FileNotFound)
		}
		return h.sendFileByUniqueID(c, fileUniqueID, user.ID)
	}

	return c.Send(buildWelcome(currentRole(c)))
}

// currentRole 从上下文取当前用户角色; 取不到时按普通用户处理(降级,不暴露额外命令)
func currentRole(c tele.Context) constants.Role {
	user, ok := RequireUser(c)
	if !ok {
		return constants.RoleUser
	}
	return constants.Role(user.Role)
}

// buildWelcome 按角色拼接欢迎消息: admin 追加管理员段, owner 在此基础上再追加 owner 段
func buildWelcome(role constants.Role) string {
	var b strings.Builder
	b.WriteString(msg.Welcome)
	if role.IsAdminOrOwner() {
		b.WriteString(msg.WelcomeAdminExtra)
	}
	if role.IsOwner() {
		b.WriteString(msg.WelcomeOwnerExtra)
	}
	return b.String()
}

// 从数据库获取文件并且将其投递到用户的聊天窗口
// 仅在 file_unique_id 和 user_id 同时匹配时才会返回记录，
// 防止deep link泄露获取到其他用户的文件
func (h *StartHandler) sendFileByUniqueID(c tele.Context, fileUniqueID string, userID uint) error {
	file, err := h.fileService.GetFileByUniqueIDAndUser(fileUniqueID, userID)
	if err != nil {
		slog.Error("deep link file lookup failed", "error", err, "file_unique_id", fileUniqueID, "user_id", userID)
		return c.Send(msg.FileSendFailed)
	}
	if file == nil {
		// File doesn't exist OR belongs to a different user — same response either way
		// to avoid leaking ownership information.
		return c.Send(msg.FileNotFound)
	}

	caption := buildFileCaption(file)

	copies, err := h.fileService.GetFileCopies(file)
	if err != nil {
		slog.Error("load file copies failed", "error", err, "file_db_id", file.ID)
	}

	sent, err := storage.SendWithFallback(c.Bot(), h.storage, c.Chat(), file, copies, caption)
	if err == nil {
		attachDeleteButton(c.Bot(), sent, file.ID)
		return nil
	}
	if errors.Is(err, storage.ErrFileUnsendable) {
		return c.Send(msg.FileUnavailable)
	}
	slog.Error("send file to user failed (deep link)", "error", err, "file_db_id", file.ID)
	return c.Send(msg.FileSendFailed)
}
