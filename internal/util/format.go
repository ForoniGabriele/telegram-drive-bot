package util

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"tg-drive-bot/internal/constants"
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
	switch constants.FileType(fileType) {
	case constants.FileTypeDocument:
		return "📄"
	case constants.FileTypeAudio:
		return "🎵"
	case constants.FileTypeVideo:
		return "🎬"
	case constants.FileTypePhoto:
		return "🖼"
	case constants.FileTypeAnimation:
		return "🎞"
	case constants.FileTypeVoice:
		return "🎤"
	case constants.FileTypeVideoNote:
		return "⏺"
	default:
		return "📎"
	}
}

// FileTypeName returns the Chinese display name for a file type.
func FileTypeName(fileType string) string {
	switch constants.FileType(fileType) {
	case constants.FileTypeDocument:
		return "文档"
	case constants.FileTypeAudio:
		return "音频"
	case constants.FileTypeVideo:
		return "视频"
	case constants.FileTypePhoto:
		return "图片"
	case constants.FileTypeAnimation:
		return "动图"
	case constants.FileTypeVoice:
		return "语音"
	case constants.FileTypeVideoNote:
		return "视频笔记"
	default:
		return "未知"
	}
}

// AllFileTypes returns all supported file type keys in display order.
func AllFileTypes() []string {
	return []string{
		constants.FileTypeDocument.String(),
		constants.FileTypeAudio.String(),
		constants.FileTypeVideo.String(),
		constants.FileTypePhoto.String(),
		constants.FileTypeAnimation.String(),
		constants.FileTypeVoice.String(),
		constants.FileTypeVideoNote.String(),
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
// 用于列表（list）/ 搜索（search）/ 随机（random）视图中的精简标题预览
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

// TruncateCaption 按 Telegram 的 caption 长度上限(UTF-16 code units)截断字符串
// Telegram 服务器对 caption 的长度计数与 JavaScript 的 String.length 一致 -- 即 UTF-16 code units
// BMP 内字符占 1 unit, 大多数 emoji(代理对) 占 2 units, 所以不能用 rune 数估算
// 不会追加 "..." -- 这是给文件 caption 用的, 调用方已经处理好提示信息, 截断标记反而干扰阅读
// 截断时避免拆开代理对, 防止产生无效的 UTF-16 序列
func TruncateCaption(s string, maxUTF16Units int) string {
	if maxUTF16Units <= 0 {
		return ""
	}
	used := 0
	for i, r := range s {
		w := 1
		if r > 0xFFFF {
			w = 2
		}
		if used+w > maxUTF16Units {
			return s[:i]
		}
		used += w
	}
	return s
}

// CaptionUTF16Len 返回字符串以 UTF-16 code units 计的长度
// 主要用于 caption 长度判断 -- 与 Telegram 服务器侧的计数方式一致
func CaptionUTF16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
