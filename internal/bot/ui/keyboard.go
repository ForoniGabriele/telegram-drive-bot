package ui

import (
	"tg-drive-bot/internal/bot/msg"

	tele "gopkg.in/telebot.v4"
)

// FileActionKeyboard 是附加到刚发送的文件消息上的单行删除键盘
func FileActionKeyboard(cat *msg.Catalog, fileID uint) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	del := m.Data(cat.BtnDelete, CBFileDel.String(), Encode(CBData{FileDBID: fileID}))
	m.Inline(m.Row(del))
	return m
}

// FileDeleteConfirmKeyboard is the confirm/cancel row shown after the first
// delete tap.
func FileDeleteConfirmKeyboard(cat *msg.Catalog, fileID uint) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	confirm := m.Data(cat.BtnConfirmDelete, CBFileDelConfirm.String(), Encode(CBData{FileDBID: fileID}))
	cancel := m.Data(cat.BtnCancel, CBFileDelCancel.String(), Encode(CBData{FileDBID: fileID}))
	m.Inline(m.Row(confirm, cancel))
	return m
}
