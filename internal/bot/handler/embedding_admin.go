package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

// EmbeddingAdminHandler 处理 owner 级别的 embedding 运维命令:
//   - /emb_re   全量重新生成 embedding(覆盖所有现有向量)
//   - /emb_sync 仅为 embedding 列为 NULL 的文件补全
//
// 两个命令共用 EmbeddingService 内部的 CAS 互斥锁,任意时刻只能跑一个批量任务
// handler 立即返回,后台 goroutine 消费 progress chan 并将进度 edit 回原消息
type EmbeddingAdminHandler struct {
	embSvc *service.EmbeddingService
}

// NewEmbeddingAdminHandler 创建一个新的 EmbeddingAdminHandler
// 当向量搜索被禁用时 embSvc 可能为 nil,handler 会拒绝命令并提示
func NewEmbeddingAdminHandler(embSvc *service.EmbeddingService) *EmbeddingAdminHandler {
	return &EmbeddingAdminHandler{embSvc: embSvc}
}

// OnEmbRe 启动全量重做 embedding 任务
func (h *EmbeddingAdminHandler) OnEmbRe(c tele.Context) error {
	return h.start(c, service.BatchModeRedo, msg.EmbBatchStartedRe)
}

// OnEmbSync 启动只补缺失 embedding 的任务
func (h *EmbeddingAdminHandler) OnEmbSync(c tele.Context) error {
	return h.start(c, service.BatchModeRefill, msg.EmbBatchStartedSync)
}

func (h *EmbeddingAdminHandler) start(c tele.Context, mode service.BatchMode, startedText string) error {
	if h.embSvc == nil {
		return c.Send("❌ 向量搜索未启用, 无法运行 embedding 批量")
	}

	sent, err := c.Bot().Send(c.Chat(), startedText)
	if err != nil {
		slog.Error("emb admin: send start message failed", "error", err, "mode", mode)
		return err
	}

	bot := c.Bot()
	chatID := c.Chat().ID
	// 使用独立 background context,避免 handler return 后被 cancel
	// 任务可能耗时数十分钟,远超 telebot handler 的生命周期
	go h.runAndReportProgress(context.Background(), bot, chatID, sent, mode)

	return nil
}

// runAndReportProgress 在后台 goroutine 中跑批量任务,并把进度持续 edit 到 sent 消息
func (h *EmbeddingAdminHandler) runAndReportProgress(ctx context.Context, bot tele.API, chatID int64, sent *tele.Message, mode service.BatchMode) {
	progressCh := make(chan service.BatchProgress, 4)

	go func() {
		err := h.embSvc.RunBatch(ctx, mode, progressCh)
		// 这里只关心 ErrBatchBusy:RunBatch 已 close(progressCh) 且未投递任何进度
		// 其他错误已通过 Phase=="failed" 的 progress 投递给消费方
		if errors.Is(err, service.ErrBatchBusy) {
			if _, sErr := bot.Send(&tele.Chat{ID: chatID}, msg.EmbBatchBusy); sErr != nil {
				slog.Warn("emb admin: send busy notice failed", "error", sErr)
			}
		}
	}()

	for p := range progressCh {
		switch p.Phase {
		case "scanning":
			// 扫描阶段不更新消息,等到 running 第一帧
		case "running":
			if p.Total == 0 {
				continue
			}
			text := fmt.Sprintf(msg.EmbBatchProgress, p.Done, p.Total, p.Phase)
			if _, err := bot.Edit(sent, text); err != nil {
				slog.Debug("emb admin: edit progress failed (likely throttled)", "error", err)
			}
		case "done":
			text := msg.EmbBatchEmpty
			if p.Total > 0 {
				text = fmt.Sprintf(msg.EmbBatchDone, p.Total)
			}
			if _, err := bot.Edit(sent, text); err != nil {
				slog.Warn("emb admin: edit done failed", "error", err)
			}
		case "failed":
			text := fmt.Sprintf(msg.EmbBatchFailed, p.Err)
			if _, err := bot.Edit(sent, text); err != nil {
				slog.Warn("emb admin: edit failed status failed", "error", err)
			}
		}
	}
}
