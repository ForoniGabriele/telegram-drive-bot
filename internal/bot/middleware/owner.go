package middleware

import (
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// OwnerOnly 仅放行 owner 角色的用户
// 非 owner(包括 admin、user)静默忽略,沿用项目"未授权不回消息"的策略
// 用于 /emb_re、/emb_sync、/cap_sync 等可能造成大量 API 调用 / 数据修改的运维命令
func OwnerOnly() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user, ok := c.Get("db_user").(*model.User)
			if !ok || !constants.Role(user.Role).IsOwner() {
				return nil
			}
			return next(c)
		}
	}
}
