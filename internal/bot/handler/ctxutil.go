package handler

import (
	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// RequireUser 从上下文中获取 Whitelist中间件所存储的白名单用户
// 如果不存在则返回 (nil, false) —— 在这种情况下，调用方应当静默返回 nil，
// 这对应了“忽略未授权用户”的策略
func RequireUser(c tele.Context) (*model.User, bool) {
	user, ok := c.Get("db_user").(*model.User)
	if !ok || user == nil {
		return nil, false
	}
	return user, true
}
