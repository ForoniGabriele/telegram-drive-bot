package ui

import (
	tele "gopkg.in/telebot.v4"
)

// FileActionKeyboard is the initial single-row "🗑 删除" keyboard attached to a freshly sent file message.
func FileActionKeyboard(fileID uint) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	del := m.Data("🗑 删除", CBFileDel.String(), Encode(CBData{FileDBID: fileID}))
	m.Inline(m.Row(del))
	return m
}

// FileDeleteConfirmKeyboard is the two-button "✅ 确认删除 / ❌ 取消" row after the first delete tap.
func FileDeleteConfirmKeyboard(fileID uint) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	confirm := m.Data("✅ 确认删除", CBFileDelConfirm.String(), Encode(CBData{FileDBID: fileID}))
	cancel := m.Data("❌ 取消", CBFileDelCancel.String(), Encode(CBData{FileDBID: fileID}))
	m.Inline(m.Row(confirm, cancel))
	return m
}
