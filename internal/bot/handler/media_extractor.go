package handler

import (
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

// 将 telebot Message 分发给相应的、针对特定类型的媒体提取器
// 当消息不包含任何受支持的媒体时返回 nil
// 顺序至关重要：某些媒体类型可以在同一条消息中共存（例如，Telegram 在历史上允许
// 照片伴随文档预览一同存在），因此我们按优先级顺序进行检查
// 优先处理内容最丰富的媒体: doc->audio->video->photo...
func extractFileInfo(m *tele.Message) *service.FileInfo {
	switch {
	case m.Document != nil:
		return extractDocument(m.Document)
	case m.Audio != nil:
		return extractAudio(m.Audio)
	case m.Video != nil:
		return extractVideo(m.Video)
	case m.Photo != nil:
		return extractPhoto(m.Photo)
	case m.Animation != nil:
		return extractAnimation(m.Animation)
	case m.Voice != nil:
		return extractVoice(m.Voice)
	case m.VideoNote != nil:
		return extractVideoNote(m.VideoNote)
	default:
		return nil
	}
}

func extractDocument(d *tele.Document) *service.FileInfo {
	return &service.FileInfo{
		FileType:     constants.FileTypeDocument.String(),
		BotFileID:    d.FileID,
		FileUniqueID: d.UniqueID,
		FileName:     d.FileName,
		MimeType:     d.MIME,
		FileSize:     d.FileSize,
	}
}

func extractAudio(a *tele.Audio) *service.FileInfo {
	return &service.FileInfo{
		FileType:     constants.FileTypeAudio.String(),
		BotFileID:    a.FileID,
		FileUniqueID: a.UniqueID,
		FileName:     a.FileName,
		MimeType:     a.MIME,
		FileSize:     a.FileSize,
		Duration:     a.Duration,
		Performer:    a.Performer,
		Title:        a.Title,
	}
}

func extractVideo(v *tele.Video) *service.FileInfo {
	return &service.FileInfo{
		FileType:     constants.FileTypeVideo.String(),
		BotFileID:    v.FileID,
		FileUniqueID: v.UniqueID,
		FileName:     v.FileName,
		MimeType:     v.MIME,
		FileSize:     v.FileSize,
		Duration:     v.Duration,
		Width:        v.Width,
		Height:       v.Height,
	}
}

func extractPhoto(p *tele.Photo) *service.FileInfo {
	return &service.FileInfo{
		FileType:     constants.FileTypePhoto.String(),
		BotFileID:    p.FileID,
		FileUniqueID: p.UniqueID,
		FileSize:     int64(p.FileSize),
		Width:        p.Width,
		Height:       p.Height,
	}
}

func extractAnimation(a *tele.Animation) *service.FileInfo {
	return &service.FileInfo{
		FileType:     constants.FileTypeAnimation.String(),
		BotFileID:    a.FileID,
		FileUniqueID: a.UniqueID,
		FileName:     a.FileName,
		MimeType:     a.MIME,
		FileSize:     a.FileSize,
		Duration:     a.Duration,
		Width:        a.Width,
		Height:       a.Height,
	}
}

func extractVoice(v *tele.Voice) *service.FileInfo {
	return &service.FileInfo{
		FileType:     constants.FileTypeVoice.String(),
		BotFileID:    v.FileID,
		FileUniqueID: v.UniqueID,
		MimeType:     v.MIME,
		FileSize:     int64(v.FileSize),
		Duration:     v.Duration,
	}
}

func extractVideoNote(v *tele.VideoNote) *service.FileInfo {
	// VideoNote is a square: Length is the side, we persist it into both Width and Height.
	return &service.FileInfo{
		FileType:     constants.FileTypeVideoNote.String(),
		BotFileID:    v.FileID,
		FileUniqueID: v.UniqueID,
		FileSize:     int64(v.FileSize),
		Duration:     v.Duration,
		Width:        v.Length,
		Height:       v.Length,
	}
}
