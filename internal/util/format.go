package util

import (
	"fmt"
	"strings"
)

// FormatFileSize formats bytes into human-readable size string.
func FormatFileSize(bytes int64) string {
	if bytes <= 0 {
		return "未知"
	}
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FileTypeIcon returns the emoji icon for a file type.
func FileTypeIcon(fileType string) string {
	switch fileType {
	case "document":
		return "📄"
	case "audio":
		return "🎵"
	case "video":
		return "🎬"
	case "photo":
		return "🖼"
	case "animation":
		return "🎞"
	case "voice":
		return "🎤"
	case "video_note":
		return "⏺"
	default:
		return "📎"
	}
}

// FileTypeName returns the Chinese display name for a file type.
func FileTypeName(fileType string) string {
	switch fileType {
	case "document":
		return "文档"
	case "audio":
		return "音频"
	case "video":
		return "视频"
	case "photo":
		return "图片"
	case "animation":
		return "动图"
	case "voice":
		return "语音"
	case "video_note":
		return "视频笔记"
	default:
		return "未知"
	}
}

// AllFileTypes returns all supported file type keys in display order.
func AllFileTypes() []string {
	return []string{
		"document", "audio", "video", "photo",
		"animation", "voice", "video_note",
	}
}

// TruncateString truncates a string to maxLen characters, appending "..." if truncated.
func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// CompactLines 将多行字符串扁平化为单行：空白行（或仅包含空格的行）会被丢弃，
// 剩余的行将用两个空格拼接
// 用于列表（list）/ 搜索（search）/ 随机（random）视图中的精简标题预览，
// 在这些视图中，原始的换行符只会增加干扰（噪点）而没有实际的有效信息
func CompactLines(s string) string {
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			kept = append(kept, trimmed)
		}
	}
	return strings.Join(kept, "  ")
}
