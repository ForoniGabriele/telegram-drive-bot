package handler

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

// StatsHandler handles the /stats command.
type StatsHandler struct {
	fileService *service.FileService
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(fileService *service.FileService) *StatsHandler {
	return &StatsHandler{fileService: fileService}
}

// OnStats handles the /stats command.
func (h *StatsHandler) OnStats(c tele.Context) error {
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}

	// Get per-type stats
	typeStats, err := h.fileService.GetStats(user.ID)
	if err != nil {
		slog.Error("get stats failed", "error", err, "user_id", user.ID)
		return c.Send(msg.StatsFetchFailed)
	}

	// Get total stats
	totalStats, err := h.fileService.GetTotalStats(user.ID)
	if err != nil {
		slog.Error("get total stats failed", "error", err, "user_id", user.ID)
		return c.Send(msg.StatsFetchFailed)
	}

	return c.Send(formatStats(typeStats, totalStats))
}

// formatStats formats the statistics message.
func formatStats(typeStats []service.FileTypeStats, totalStats *service.TotalStats) string {
	var sb strings.Builder
	sb.WriteString("📊 你的文件库统计\n\n")

	// Build a map for quick lookup
	statsMap := make(map[string]service.FileTypeStats)
	for _, s := range typeStats {
		statsMap[s.FileType] = s
	}

	// Display all types in order
	for _, ft := range util.AllFileTypes() {
		icon := util.FileTypeIcon(ft)
		name := util.FileTypeName(ft)
		stat, ok := statsMap[ft]
		count := int64(0)
		if ok {
			count = stat.Count
		}
		sb.WriteString(fmt.Sprintf("%s %s: %d 个\n", icon, name, count))
	}

	// Total summary
	sb.WriteString(fmt.Sprintf("\n📦 总计: %d 个文件\n", totalStats.TotalCount))
	sb.WriteString(fmt.Sprintf("💾 总大小: %s\n", util.FormatFileSize(totalStats.TotalSize)))

	if totalStats.Earliest != nil && *totalStats.Earliest != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", (*totalStats.Earliest)[:19]); err == nil {
			sb.WriteString(fmt.Sprintf("📅 最早存入: %s\n", t.Format(constants.DateDay)))
		}
	}
	if totalStats.Latest != nil && *totalStats.Latest != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", (*totalStats.Latest)[:19]); err == nil {
			sb.WriteString(fmt.Sprintf("📅 最近存入: %s\n", t.Format(constants.DateDay)))
		}
	}

	return sb.String()
}
