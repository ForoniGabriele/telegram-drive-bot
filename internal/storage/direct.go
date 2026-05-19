package storage

import (
	"fmt"

	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// Direct 模式通过使用 Telegram 的 file_id 直接发送文件
type Direct struct{}

func NewDirect() *Direct {
	return &Direct{}
}

// 在 direct 模式下不执行任何操作, 空切片
func (d *Direct) ForwardToStorage(bot tele.API, msg *tele.Message) ([]model.CopyLocation, error) {
	return nil, nil
}

// 通过 file_id 将文件发送给用户，并返回发送的消息
// 在 direct 模式下会忽略 copies 参数
func (d *Direct) SendFileToUser(bot tele.API, userChat *tele.Chat, file *model.File, _ []model.CopyLocation, caption string) (*tele.Message, error) {
	sendable := BuildSendable(file, caption)
	if sendable == nil {
		return nil, fmt.Errorf("unsupported file type for direct send: %s", file.FileType)
	}

	sent, err := bot.Send(userChat, sendable)
	if err != nil {
		return nil, fmt.Errorf("direct send file to user failed: %w", err)
	}
	return sent, nil
}

// 在 direct 模式下不执行任何操作
func (d *Direct) DeleteFromStorage(_ tele.API, _ []model.CopyLocation) error {
	return nil
}

// 根据 File 模型构建相应的 telebot Sendable 对象
// caption 参数会覆盖文件原始的caption
func BuildSendable(file *model.File, caption string) tele.Sendable {
	switch constants.FileType(file.FileType) {
	case constants.FileTypeDocument:
		return &tele.Document{
			File:     tele.File{FileID: file.BotFileID},
			FileName: file.FileName,
			MIME:     file.MimeType,
			Caption:  caption,
		}
	case constants.FileTypeAudio:
		return &tele.Audio{
			File:      tele.File{FileID: file.BotFileID},
			FileName:  file.FileName,
			MIME:      file.MimeType,
			Duration:  file.Duration,
			Performer: file.Performer,
			Title:     file.Title,
			Caption:   caption,
		}
	case constants.FileTypeVideo:
		return &tele.Video{
			File:     tele.File{FileID: file.BotFileID},
			FileName: file.FileName,
			MIME:     file.MimeType,
			Duration: file.Duration,
			Width:    file.Width,
			Height:   file.Height,
			Caption:  caption,
		}
	case constants.FileTypePhoto:
		return &tele.Photo{
			File:    tele.File{FileID: file.BotFileID},
			Width:   file.Width,
			Height:  file.Height,
			Caption: caption,
		}
	case constants.FileTypeAnimation:
		return &tele.Animation{
			File:     tele.File{FileID: file.BotFileID},
			FileName: file.FileName,
			MIME:     file.MimeType,
			Duration: file.Duration,
			Width:    file.Width,
			Height:   file.Height,
			Caption:  caption,
		}
	case constants.FileTypeVoice:
		return &tele.Voice{
			File:     tele.File{FileID: file.BotFileID},
			MIME:     file.MimeType,
			Duration: file.Duration,
			Caption:  caption,
		}
	case constants.FileTypeVideoNote:
		return &tele.VideoNote{
			File:     tele.File{FileID: file.BotFileID},
			Duration: file.Duration,
			Length:   file.Width,
		}
	default:
		return nil
	}
}
