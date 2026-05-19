package ui

import (
	"errors"

	tele "gopkg.in/telebot.v4"
)

// 在 isEdit 为 true 时执行 c.Edit，否则执行 c.Send
// 它会忽略 Telegram 返回的 "message is not modified"（消息未修改） / "same content"（内容相同）的 400 错误:
// 当新文本 + 回复标记（reply markup）与当前消息在字节上完全一致时，Telegram 会返回这个错误 ——
// 例如：用户点击了当前已经处于激活状态的类型过滤器。在这种情况下，我们无需执行任何操作；
// 如果将此错误作为通用的处理程序（handler）错误呈现给用户，将会产生误导
// 我们仍然返回 nil，以便调用方正常的回应/响应流程可以继续

func EditOrSend(c tele.Context, text string, markup *tele.ReplyMarkup, isEdit bool) error {
	var err error
	if isEdit {
		err = c.Edit(text, markup, tele.ModeHTML)
	} else {
		err = c.Send(text, markup, tele.ModeHTML)
	}
	if err != nil && (errors.Is(err, tele.ErrSameMessageContent) || errors.Is(err, tele.ErrMessageNotModified)) {
		return nil
	}
	return err
}
