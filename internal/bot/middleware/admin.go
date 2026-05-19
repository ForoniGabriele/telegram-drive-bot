package middleware

import (
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"

	tele "gopkg.in/telebot.v4"
)

// AdminOnly returns a middleware that restricts access to admin and owner roles.
// Non-admin users are silently ignored.
func AdminOnly() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user, ok := c.Get("db_user").(*model.User)
			if !ok || !constants.Role(user.Role).IsAdminOrOwner() {
				return nil // Silently ignore non-admins
			}
			return next(c)
		}
	}
}
