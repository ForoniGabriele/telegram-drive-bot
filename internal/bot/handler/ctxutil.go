package handler

import (
	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// RequireUser 从上下文中获取 Whitelist中间件所存储的白名单用户
// 如果不存在则返回 (nil, false)不给白名单外的用户回复
func RequireUser(c tele.Context) (*model.User, bool) {
	user, ok := c.Get("db_user").(*model.User)
	if !ok || user == nil {
		return nil, false
	}
	return user, true
}
