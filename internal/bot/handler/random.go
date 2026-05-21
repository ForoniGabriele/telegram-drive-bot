package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/storage"
	"tg-drive-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

// Random command bounds.
const (
	randDefaultCount    = 5
	randMaxCount        = 10
	randSendInterval    = 100 * time.Millisecond
	randCaptionMaxChars = 100 // rune-length cap applied to the file's original caption
)

// Type sets for each random command. Kept as string slices because the repository
// layer filters with a SQL IN (...) clause.
var (
	randAllMediaTypes = []string{
		constants.FileTypePhoto.String(),
		constants.FileTypeVideo.String(),
		constants.FileTypeAnimation.String(),
		constants.FileTypeVideoNote.String(),
	}
	randVideoTypes = []string{
		constants.FileTypeVideo.String(),
		constants.FileTypeAnimation.String(),
	}
	randPhotoTypes = []string{
		constants.FileTypePhoto.String(),
	}
)

// RandomHandler handles /rand, /randv, /randp commands.
type RandomHandler struct {
	fileService *service.FileService
	storage     storage.Storage
}

// NewRandomHandler creates a new RandomHandler.
func NewRandomHandler(fileService *service.FileService, store storage.Storage) *RandomHandler {
	return &RandomHandler{
		fileService: fileService,
		storage:     store,
	}
}

// OnRand handles /rand — random visual media (photo/video/animation/video_note).
func (h *RandomHandler) OnRand(c tele.Context) error {
	return h.handle(c, "/rand", randAllMediaTypes)
}

// OnRandVideo handles /randv — random video + animation.
func (h *RandomHandler) OnRandVideo(c tele.Context) error {
	return h.handle(c, "/randv", randVideoTypes)
}

// OnRandPhoto handles /randp — random photos.
func (h *RandomHandler) OnRandPhoto(c tele.Context) error {
	return h.handle(c, "/randp", randPhotoTypes)
}

// handle is the shared flow: parse count, query random files, send each in turn.
func (h *RandomHandler) handle(c tele.Context, cmdName string, fileTypes []string) error {
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}

	count, err := parseRandCount(c.Message().Payload)
	if err != nil {
		return c.Send(fmt.Sprintf(msg.RandInvalidNumber, cmdName, randMaxCount))
	}

	files, err := h.fileService.GetRandomFiles(user.ID, fileTypes, count)
	if err != nil {
		slog.Error("get random files failed", "error", err, "user_id", user.ID, "types", fileTypes, "count", count)
		return c.Send(msg.RandFetchFailed)
	}
	if len(files) == 0 {
		return c.Send(msg.RandNoMedia)
	}

	if len(files) < count {
		if err := c.Send(fmt.Sprintf(msg.RandPartial, len(files))); err != nil {
			slog.Warn("send partial notice failed", "error", err, "user_id", user.ID)
		}
	}

	for i := range files {
		h.sendOne(c, &files[i])
		if i < len(files)-1 {
			time.Sleep(randSendInterval)
		}
	}
	return nil
}

// 使用配置的存储服务发送单个随机文件，其采用与 /list 的文件单文件获取回调相同的"直接发送降级模式（direct-send fallback）"
// 发送失败只记录日志，而不会中断批量任务, 用户仍会收到那些发送成功的文件
func (h *RandomHandler) sendOne(c tele.Context, file *model.File) {
	copies, err := h.fileService.GetFileCopies(file)
	if err != nil {
		slog.Error("load file copies failed", "error", err, "file_db_id", file.ID)
	}

	caption := buildRandomCaption(file)

	sent, err := storage.SendWithFallback(c.Bot(), h.storage, c.Chat(), file, copies, caption)
	if err == nil {
		attachDeleteButton(c.Bot(), sent, file.ID)
		return
	}
	if !errors.Is(err, storage.ErrFileUnsendable) {
		slog.Error("send random file failed", "error", err, "file_db_id", file.ID)
	}
	// ErrFileUnsendable 已由 SendWithFallback 内部 log 过, 不再重复
}

// buildRandomCaption 用于构建附加在 /rand 回复中的简短caption
// 第一行为文件名，下一行为截断为 randCaptionMaxChars（100）个字符的原始注释
// 当两者都为空时返回 ""
// 视频便签（video_note）会被跳过，因为 Telegram 不允许为视频便签添加说明
// （BuildSendable 也会在处理该类型时将其丢弃，因此这里只是做防御性编程）
func buildRandomCaption(file *model.File) string {
	if file.FileType == constants.FileTypeVideoNote.String() {
		return ""
	}

	var parts []string
	if name := strings.TrimSpace(file.FileName); name != "" {
		parts = append(parts, "📎 "+name)
	}
	if orig := strings.TrimSpace(file.Caption); orig != "" {
		truncated := util.TruncateString(orig, randCaptionMaxChars)
		parts = append(parts, "📝 "+util.CompactLines(truncated))
	}
	return strings.Join(parts, "\n")
}

// parseRandCount parses the /rand [n] payload. Empty payload returns the default.
// Returns an error for non-numeric input; values are clamped to [1, randMaxCount].
func parseRandCount(payload string) (int, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return randDefaultCount, nil
	}
	n, err := strconv.Atoi(payload)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		n = 1
	}
	if n > randMaxCount {
		n = randMaxCount
	}
	return n, nil
}
