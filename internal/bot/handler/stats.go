package handler

import (
	"fmt"
	"log/slog"
	"strings"

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
	cat := Cat(c)
	lang := Lang(c)
	user, ok := RequireUser(c)
	if !ok {
		return nil
	}

	// Get per-type stats
	typeStats, err := h.fileService.GetStats(user.ID)
	if err != nil {
		slog.Error("get stats failed", "error", err, "user_id", user.ID)
		return c.Send(cat.StatsFetchFailed)
	}

	// Get total stats
	totalStats, err := h.fileService.GetTotalStats(user.ID)
	if err != nil {
		slog.Error("get total stats failed", "error", err, "user_id", user.ID)
		return c.Send(cat.StatsFetchFailed)
	}

	return c.Send(formatStats(cat, lang, typeStats, totalStats))
}

// formatStats formats the statistics message. The numeric labels around per-type
// counts and totals are short and language-neutral enough that we keep them
// inline rather than threading another half-dozen Catalog fields through.
func formatStats(cat *msg.Catalog, lang string, typeStats []service.FileTypeStats, totalStats *service.TotalStats) string {
	var sb strings.Builder
	sb.WriteString(cat.StatsBlock)

	// Build a map for quick lookup
	statsMap := make(map[string]service.FileTypeStats)
	for _, s := range typeStats {
		statsMap[s.FileType] = s
	}

	// Display all types in order
	for _, ft := range util.AllFileTypes() {
		icon := util.FileTypeIcon(ft)
		name := util.FileTypeName(lang, ft)
		stat, ok := statsMap[ft]
		count := int64(0)
		if ok {
			count = stat.Count
		}
		sb.WriteString(fmt.Sprintf("%s %s: %d\n", icon, name, count))
	}

	// Total summary. "Total" uses the localized type-name "All" placeholder
	// label only when it adds value; here we just lean on the emoji + a colon.
	sb.WriteString(fmt.Sprintf("\n📦 %s: %d\n", cat.BtnFilterAll, totalStats.TotalCount))
	sb.WriteString(fmt.Sprintf("💾 %s\n", util.FormatFileSize(totalStats.TotalSize)))

	if totalStats.Earliest != nil {
		sb.WriteString(fmt.Sprintf("📅 %s\n", totalStats.Earliest.Format(constants.DateDay)))
	}
	if totalStats.Latest != nil {
		sb.WriteString(fmt.Sprintf("📅 %s\n", totalStats.Latest.Format(constants.DateDay)))
	}

	return sb.String()
}
