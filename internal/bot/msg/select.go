package msg

import "strings"

// 目前支持的语言
const (
	LangZh = "zh"
	LangEn = "en"
)

// For 返回给定的语言标签对应的Catalog，标签无法识别或为空时回退到Zh
func For(lang string) *Catalog {
	switch lang {
	case LangEn:
		return &En
	default:
		return &Zh
	}
}

// Normalize 将 Telegram 上报的 language_code（例如 "zh-hans"、
// "zh-CN"、"en-US"、"ru"）映射为支持的语言标签之一.
// 任何中文变体都会被归并（折叠）为 "zh"；其他所有语言则全部回退到"en"
//
// 输入为空时返回 ""，以便调用者能够区分“没有语言提示”与用户的“明确选择”

func Normalize(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return ""
	}
	if code == "zh" || strings.HasPrefix(code, "zh-") || strings.HasPrefix(code, "zh_") {
		return LangZh
	}
	return LangEn
}

// 是否是支持的语言
func IsSupported(lang string) bool {
	return lang == LangZh || lang == LangEn
}
