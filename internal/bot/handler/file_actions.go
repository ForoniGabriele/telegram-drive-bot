package handler

import (
	"log/slog"

	"tg-drive-bot/internal/bot/ui"

	tele "gopkg.in/telebot.v4"
)

// attachDeleteButton 把"🗑 删除"按钮附加到一条已发送的文件消息上
func attachDeleteButton(bot tele.API, m *tele.Message, fileDBID uint) {
	if m == nil {
		return
	}
	if m.Chat == nil {
		slog.Warn("attach delete button skipped: sent message has no Chat", "file_db_id", fileDBID)
		return
	}
	if _, err := bot.EditReplyMarkup(m, ui.FileActionKeyboard(fileDBID)); err != nil {
		slog.Warn("attach delete button failed", "error", err, "file_db_id", fileDBID)
	}
}
