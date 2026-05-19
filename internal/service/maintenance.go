package service

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"tg-drive-bot/internal/repository"
)

// MaintenanceService 承载 owner 级别的运维操作:目前是媒体组 caption 回填(/cap_sync)
// 未来如果增加更多跨表批量任务,都放在这里
//
// 每种批量任务用自己的 atomic.Bool 互斥(避免并发触发同一种任务)
// 不同任务之间互不阻塞 -- cap_sync 跑的时候可以同时跑 emb_sync
type MaintenanceService struct {
	repo           repository.MaintenanceRepository
	capSyncRunning atomic.Bool
}

// NewMaintenanceService 创建一个 MaintenanceService
func NewMaintenanceService(repo repository.MaintenanceRepository) *MaintenanceService {
	return &MaintenanceService{repo: repo}
}

// ErrCapSyncBusy 在已有 cap_sync 任务运行时返回
var ErrCapSyncBusy = errors.New("cap_sync already running")

// CapSyncResult 透传 repository 层的同名结构
type CapSyncResult = repository.CapSyncResult

// RunCaptionSync 触发一次 caption 同步
// 同一时刻只能跑一个:并发触发立即返回 ErrCapSyncBusy
// 内部 SQL 在单个事务里完成,执行时间通常 < 1 秒(取决于 media_group 数量),不需要进度回调
// ctx 当前未被 repository 层消费,但保留参数位以便后续接入超时控制
func (s *MaintenanceService) RunCaptionSync(_ context.Context) (CapSyncResult, error) {
	if !s.capSyncRunning.CompareAndSwap(false, true) {
		return CapSyncResult{}, ErrCapSyncBusy
	}
	defer s.capSyncRunning.Store(false)

	result, err := s.repo.CaptionSync()
	if err != nil {
		slog.Error("cap_sync failed", "error", err)
		return CapSyncResult{}, err
	}
	slog.Info("cap_sync completed",
		"files_updated", result.FilesUpdated,
		"messages_updated", result.MessagesUpdated)
	return result, nil
}
