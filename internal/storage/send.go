package storage

import (
	"errors"
	"fmt"
	"log/slog"

	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// ErrFileUnsendable 表示存储层与 file_id 直发都无法把文件投递给用户
// 调用方据此回退到"暂时不可用"提示
var ErrFileUnsendable = errors.New("file unsendable: storage failed and direct send unavailable")

// SendWithFallback 用配置好的 Storage 发文件; 当 Channel 模式所有副本失败时,
// 自动用 file_id 直发兜底, 让用户仍能拿到文件
//
// 返回的 *tele.Message 是用户最终收到的那条消息(供调用方挂删除按钮等)
// 当存储路径返回非 ErrAllCopiesFailed 的错误时, 直接透传该错误 -- 这类错误
// (网络抖动、权限错误等)不应该静默用 file_id 兜底, 让上层决定怎么提示用户
// 当 channel 全失败且 file_id 也无法直发时, 返回 ErrFileUnsendable
func SendWithFallback(bot tele.API, store Storage, chat *tele.Chat, file *model.File, copies []model.CopyLocation, caption string) (*tele.Message, error) {
	sent, err := store.SendFileToUser(bot, chat, file, copies, caption)
	if err == nil {
		return sent, nil
	}
	if !errors.Is(err, ErrAllCopiesFailed) {
		return nil, err
	}

	// All channel copies failed — fall back to a direct file_id send.
	slog.Warn("all storage copies failed, falling back to direct send", "file_db_id", file.ID)
	sendable := BuildSendable(file, caption)
	if sendable == nil {
		return nil, fmt.Errorf("%w: unsupported file type %s", ErrFileUnsendable, file.FileType)
	}
	fallback, sendErr := bot.Send(chat, sendable)
	if sendErr != nil {
		slog.Error("direct-send fallback failed", "error", sendErr, "file_db_id", file.ID)
		return nil, fmt.Errorf("%w: %v", ErrFileUnsendable, sendErr)
	}
	return fallback, nil
}
