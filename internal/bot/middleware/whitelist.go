package middleware

import (
	"tg-drive-bot/internal/service"

	tele "gopkg.in/telebot.v4"
)

// Whitelist returns a middleware that checks if the sender is in the user whitelist.
// Unauthorized users are silently ignored (no response).
func Whitelist(userService *service.UserService) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender == nil {
				return nil // No sender, ignore
			}

			user, err := userService.GetByTelegramID(sender.ID)
			if err != nil || user == nil {
				return nil // Not whitelisted, silently ignore
			}

			// Update user info from Telegram if changed
			_ = userService.UpdateUserInfo(user, sender.Username, sender.FirstName)

			// Store user in context for downstream handlers
			c.Set("db_user", user)
			return next(c)
		}
	}
}
