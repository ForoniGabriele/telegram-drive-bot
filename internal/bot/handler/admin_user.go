package handler

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"tg-drive-bot/internal/bot/msg"
	"tg-drive-bot/internal/bot/ui"
	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"
	"tg-drive-bot/internal/service"
	"tg-drive-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

// AdminUserHandler handles admin user management commands.
type AdminUserHandler struct {
	userService *service.UserService
}

// NewAdminUserHandler creates a new AdminUserHandler.
func NewAdminUserHandler(userService *service.UserService) *AdminUserHandler {
	return &AdminUserHandler{userService: userService}
}

const userListPageSize = 8

// OnAddUser handles the /adduser command.
func (h *AdminUserHandler) OnAddUser(c tele.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send(msg.AddUserUsage)
	}

	telegramID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send(msg.ErrInvalidID)
	}

	status, err := h.userService.AddUser(telegramID)
	if err != nil {
		slog.Error("add user failed", "error", err, "telegram_id", telegramID)
		return c.Send(msg.AddUserFailed)
	}

	switch status {
	case "already_exists":
		return c.Send(fmt.Sprintf("⚠️ 用户 %d 已在白名单中", telegramID))
	case "created":
		return c.Send(fmt.Sprintf("✅ 已添加用户 %d 到白名单", telegramID))
	default:
		return c.Send(msg.ErrUnknown)
	}
}

// OnRemoveUser handles the /removeuser command.
func (h *AdminUserHandler) OnRemoveUser(c tele.Context) error {
	operator, ok := RequireUser(c)
	if !ok {
		return nil
	}

	args := c.Args()
	if len(args) == 0 {
		return c.Send(msg.RemoveUserUsage)
	}

	telegramID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send(msg.ErrInvalidID)
	}

	status, err := h.userService.RemoveUser(telegramID, constants.Role(operator.Role))
	if err != nil {
		slog.Error("remove user failed", "error", err, "telegram_id", telegramID, "operator_id", operator.ID)
		return c.Send(msg.RemoveUserFailed)
	}

	switch status {
	case "not_found":
		return c.Send(fmt.Sprintf("❌ 用户 %d 不在白名单中", telegramID))
	case "cannot_remove_owner":
		return c.Send(msg.CannotRemoveOwner)
	case "only_owner_can_remove_admin":
		return c.Send(msg.OnlyOwnerRemoveAdmin)
	case "removed":
		return c.Send(fmt.Sprintf("✅ 已将用户 %d 从白名单移除", telegramID))
	default:
		return c.Send(msg.ErrUnknown)
	}
}

// OnListUser handles the /listuser command.
func (h *AdminUserHandler) OnListUser(c tele.Context) error {
	return h.sendUserListPage(c, 1, false)
}

// OnUserListCallback handles user list pagination.
func (h *AdminUserHandler) OnUserListCallback(c tele.Context) error {
	data, err := ui.Decode(c.Callback().Data)
	if err != nil {
		return nil
	}
	page := data.Page
	if page < 1 {
		page = 1
	}
	return h.sendUserListPage(c, page, true)
}

// OnUserInfoCallback handles user info display.
func (h *AdminUserHandler) OnUserInfoCallback(c tele.Context) error {
	operator, ok := RequireUser(c)
	if !ok {
		return nil
	}

	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.UserID == 0 {
		return nil
	}

	user, err := h.userService.GetByID(data.UserID)
	if err != nil || user == nil {
		return c.RespondText(msg.UserNotFound)
	}

	fileCount, _ := h.userService.CountFilesByUserID(user.ID)

	return h.sendUserInfo(c, user, operator, fileCount)
}

// OnUserDeleteCallback handles user deletion.
func (h *AdminUserHandler) OnUserDeleteCallback(c tele.Context) error {
	operator, ok := RequireUser(c)
	if !ok {
		return nil
	}

	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.TelegramID == 0 {
		return nil
	}

	status, err := h.userService.RemoveUser(data.TelegramID, constants.Role(operator.Role))
	if err != nil {
		slog.Error("delete user failed", "error", err, "telegram_id", data.TelegramID, "operator_id", operator.ID)
		return c.RespondText(msg.DeleteUserFailed)
	}

	switch status {
	case "cannot_remove_owner":
		return c.RespondText(msg.CannotRemoveOwner)
	case "only_owner_can_remove_admin":
		return c.RespondText(msg.OnlyOwnerRemoveAdmin)
	case "removed":
		return h.sendUserListPage(c, 1, true)
	default:
		return c.RespondText(msg.ErrOperation)
	}
}

// OnUserPromoteCallback handles user promotion to admin.
func (h *AdminUserHandler) OnUserPromoteCallback(c tele.Context) error {
	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.UserID == 0 {
		return nil
	}

	status, err := h.userService.PromoteUser(data.UserID)
	if err != nil {
		slog.Error("promote user failed", "error", err, "user_id", data.UserID)
		return c.RespondText(msg.PromoteFailed)
	}

	switch status {
	case "cannot_modify_owner":
		return c.RespondText(msg.CannotModifyOwner)
	case "already_admin":
		return c.RespondText(msg.AlreadyAdmin)
	case "promoted":
		return c.RespondText(msg.Promoted)
	default:
		return c.RespondText(msg.ErrOperation)
	}
}

// OnUserDemoteCallback handles user demotion from admin.
func (h *AdminUserHandler) OnUserDemoteCallback(c tele.Context) error {
	operator, ok := RequireUser(c)
	if !ok {
		return nil
	}

	data, err := ui.Decode(c.Callback().Data)
	if err != nil || data.UserID == 0 {
		return nil
	}

	status, err := h.userService.DemoteUser(data.UserID, constants.Role(operator.Role))
	if err != nil {
		slog.Error("demote user failed", "error", err, "user_id", data.UserID, "operator_id", operator.ID)
		return c.RespondText(msg.DemoteFailed)
	}

	switch status {
	case "cannot_modify_owner":
		return c.RespondText(msg.CannotModifyOwner)
	case "only_owner_can_demote":
		return c.RespondText(msg.OnlyOwnerDemoteAdmin)
	case "already_user":
		return c.RespondText(msg.AlreadyUser)
	case "demoted":
		return c.RespondText(msg.Demoted)
	default:
		return c.RespondText(msg.ErrOperation)
	}
}

// sendUserListPage sends or edits the user list.
func (h *AdminUserHandler) sendUserListPage(c tele.Context, page int, isEdit bool) error {
	users, total, err := h.userService.ListUsers(page, userListPageSize)
	if err != nil {
		slog.Error("list users failed", "error", err, "page", page)
		return c.Send(msg.UserListFetchFailed)
	}

	pag := &util.Pagination{Page: page, PageSize: userListPageSize, Total: total}

	// Build text
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 用户管理 (共 %d 人)\n\n", total))
	for i, u := range users {
		num := (page-1)*userListPageSize + i + 1
		icon := service.RoleIcon(u.Role)
		name := service.DisplayName(&u)
		roleName := service.FormatRole(u.Role)
		sb.WriteString(fmt.Sprintf("%d. %s %s (ID: %d) - %s\n", num, icon, name, u.TelegramID, roleName))
	}

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	// User detail buttons, 2 per row.
	var userBtns []tele.Btn
	for _, u := range users {
		icon := service.RoleIcon(u.Role)
		name := service.DisplayName(&u)
		userBtns = append(userBtns, markup.Data(
			fmt.Sprintf("%s %s", name, icon),
			ui.CBUserInfo.String(),
			ui.Encode(ui.CBData{UserID: u.ID}),
		))
	}
	for i := 0; i < len(userBtns); i += 2 {
		end := i + 2
		if end > len(userBtns) {
			end = len(userBtns)
		}
		rows = append(rows, markup.Row(userBtns[i:end]...))
	}

	rows = append(rows, ui.PaginationRow(markup, ui.CBUserList, pag, func(p int) string {
		return ui.Encode(ui.CBData{Page: p})
	}))

	markup.Inline(rows...)

	return ui.EditOrSend(c, sb.String(), markup, isEdit)
}

// sendUserInfo sends user detail info with action buttons.
func (h *AdminUserHandler) sendUserInfo(c tele.Context, user, operator *model.User, fileCount int64) error {
	icon := service.RoleIcon(user.Role)
	name := service.DisplayName(user)
	roleName := service.FormatRole(user.Role)

	text := fmt.Sprintf(
		"👤 用户信息\n━━━━━━━━━━━━━━━\n📛 用户名: %s %s\n🆔 Telegram ID: %d\n🔑 角色: %s\n📦 文件数: %d\n📅 加入时间: %s",
		name, icon, user.TelegramID, roleName, fileCount, user.CreatedAt.Format(constants.DateDay),
	)

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	targetRole := constants.Role(user.Role)
	operatorRole := constants.Role(operator.Role)

	// Action buttons are shown only when the operator is allowed to manage the target.
	if !targetRole.IsOwner() {
		canOperate := true
		if targetRole == constants.RoleAdmin && !operatorRole.IsOwner() {
			canOperate = false
		}

		if canOperate {
			rows = append(rows, markup.Row(
				markup.Data("🗑 删除用户", ui.CBUserDel.String(), ui.Encode(ui.CBData{TelegramID: user.TelegramID})),
			))

			switch targetRole {
			case constants.RoleUser:
				rows = append(rows, markup.Row(
					markup.Data("⬆️ 提升为管理员", ui.CBUserPromote.String(), ui.Encode(ui.CBData{UserID: user.ID})),
				))
			case constants.RoleAdmin:
				rows = append(rows, markup.Row(
					markup.Data("⬇️ 降级为普通用户", ui.CBUserDemote.String(), ui.Encode(ui.CBData{UserID: user.ID})),
				))
			}
		}
	}

	rows = append(rows, markup.Row(
		markup.Data("◀ 返回用户列表", ui.CBUserList.String(), ui.Encode(ui.CBData{Page: 1})),
	))

	markup.Inline(rows...)
	return ui.EditOrSend(c, text, markup, true)
}
