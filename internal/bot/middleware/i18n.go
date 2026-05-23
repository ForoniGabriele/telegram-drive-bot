package middleware

import (
	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// I18n 为当前 context 选取对应的i18n Catalog，并将其以 "catalog" 键名存储在 telebot 上下文中
//
// 语言解析顺序：
//  1. 白名单用户所存储的 User.Language（通过 /lang 命令设置）
//  2. 发送者所上报的 Telegram 客户端 language_code 语言代码
//  3. msg.LangZh（默认中文，与项目最开始的版本相匹配）
func I18n() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			lang := resolveLang(c)
			c.Set("lang", lang)
			c.Set("catalog", msg.For(lang))
			return next(c)
		}
	}
}

func resolveLang(c tele.Context) string {
	if u, ok := c.Get("db_user").(*model.User); ok && u != nil && msg.IsSupported(u.Language) {
		return u.Language
	}
	if sender := c.Sender(); sender != nil {
		if norm := msg.Normalize(sender.LanguageCode); norm != "" {
			return norm
		}
	}
	return msg.LangZh
}
