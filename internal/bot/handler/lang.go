package handler

import (
	"fmt"
	"log/slog"
	"strings"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

// LangHandler handles the /lang command for viewing and switching the
// caller's preferred language.
type LangHandler struct {
	userService *service.UserService
}

// NewLangHandler creates a new LangHandler.
func NewLangHandler(userService *service.UserService) *LangHandler {
	return &LangHandler{userService: userService}
}

// OnLang handles `/lang`, `/lang zh`, `/lang en`.
//
// 缓存延迟说明:Whitelist 中间件对 *model.User 做 30s TTL 缓存,所以
// 切换语言后,如果该用户在 30s 内继续发命令,middleware 拿到的仍是旧
// User.Language 字段(旧 catalog).这与现有"角色变更 30s 内生效"的设计一致
// 私人 bot 场景下可以接受
func (h *LangHandler) OnLang(c tele.Context) error {
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}
	cat := Cat(c)

	arg := strings.ToLower(strings.TrimSpace(c.Message().Payload))
	if arg == "" {
		current := Lang(c)
		return c.Send(fmt.Sprintf(cat.LangCurrent, current))
	}

	if !msg.IsSupported(arg) {
		return c.Send(cat.LangInvalid)
	}

	if err := h.userService.UpdateLanguage(user, arg); err != nil {
		slog.Error("update language failed", "error", err, "user_id", user.ID, "lang", arg)
		return c.Send(cat.ErrOperation)
	}

	// 返回当前使用的语言消息
	newCat := msg.For(arg)
	return c.Send(fmt.Sprintf(newCat.LangSet, arg))
}
