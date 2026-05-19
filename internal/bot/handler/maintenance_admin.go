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

// MaintenanceAdminHandler 处理 owner 级别的运维命令(目前只有 /cap_sync)
// 与 EmbeddingAdminHandler 不同,cap_sync 是纯 SQL 任务,执行时间通常 < 1 秒,
// 不需要后台 goroutine + 进度回调,handler 内同步执行后回 1 条完成消息
type MaintenanceAdminHandler struct {
	maintSvc *service.MaintenanceService
}

// NewMaintenanceAdminHandler 创建一个 MaintenanceAdminHandler
func NewMaintenanceAdminHandler(maintSvc *service.MaintenanceService) *MaintenanceAdminHandler {
	return &MaintenanceAdminHandler{maintSvc: maintSvc}
}

// OnCapSync 处理 /cap_sync:为同 media_group 的文件/消息回填空 caption
func (h *MaintenanceAdminHandler) OnCapSync(c tele.Context) error {
	if err := c.Send(msg.CapSyncStarted); err != nil {
		slog.Warn("cap_sync: send start notice failed", "error", err)
	}

	result, err := h.maintSvc.RunCaptionSync(context.Background())
	if err != nil {
		if errors.Is(err, service.ErrCapSyncBusy) {
			return c.Send(msg.CapSyncBusy)
		}
		return c.Send(fmt.Sprintf(msg.CapSyncFailed, err))
	}

	return c.Send(fmt.Sprintf(msg.CapSyncDone, result.FilesUpdated, result.MessagesUpdated))
}
