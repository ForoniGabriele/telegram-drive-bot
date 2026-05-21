// Package ui 封装了 Telegram UI 相关的处理：callback data编码、
// 键盘构建以及分页列表的渲染
package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CBKind is the telebot InlineButton.Unique value, which routes a callback to a handler.
type CBKind string

const (
	CBList           CBKind = "list"
	CBSearch         CBKind = "srch"
	CBFile           CBKind = "file"
	CBFileDel        CBKind = "fdel"
	CBFileDelConfirm CBKind = "fdelc"
	CBFileDelCancel  CBKind = "fdelx"
	CBUserList       CBKind = "ulist"
	CBUserInfo       CBKind = "uinfo"
	CBUserDel        CBKind = "udel"
	CBUserPromote    CBKind = "uprom"
	CBUserDemote     CBKind = "udem"
	CBNoop           CBKind = "noop"
)

func (k CBKind) String() string { return string(k) }

// ErrCBMalformed signals callback data that cannot be decoded.
var ErrCBMalformed = errors.New("malformed callback data")

// CBData 保存了从 InlineButton.Data 携带的解码后的playload数据, 所有字段均为可选
type CBData struct {
	FileType   string // "all", "document", ... used by list/search
	Page       int    // pagination target
	FileDBID   uint   // file row id (数据库主键,与 Telegram bot file_id 是两回事)
	UserID     uint   // user row id (admin flows)
	TelegramID int64  // raw telegram user id (admin delete)
	CacheKey   string // search-context cache key
}

// Encode 将 CBData 序列化为 "k=v|k=v" 格式，空字段会被省略
// 在实际输入情况下，能确保始终控制在 Telegram 64 字节的 callback_data 长度限制内
func Encode(d CBData) string {
	var parts []string
	if d.FileType != "" {
		parts = append(parts, "t="+d.FileType)
	}
	if d.Page != 0 {
		parts = append(parts, "p="+strconv.Itoa(d.Page))
	}
	if d.FileDBID != 0 {
		parts = append(parts, "f="+strconv.FormatUint(uint64(d.FileDBID), 10))
	}
	if d.UserID != 0 {
		parts = append(parts, "u="+strconv.FormatUint(uint64(d.UserID), 10))
	}
	if d.TelegramID != 0 {
		parts = append(parts, "tg="+strconv.FormatInt(d.TelegramID, 10))
	}
	if d.CacheKey != "" {
		parts = append(parts, "k="+d.CacheKey)
	}
	return strings.Join(parts, "|")
}

// Decode 将 "k=v|k=v" 解析为 CBData. 空字符串会直接返回零值的 CBData 且不返回错误，
// 这样接收方可以将“不存在”和“存在但为空”这两种情况等同看待
func Decode(s string) (CBData, error) {
	var d CBData
	if s == "" {
		return d, nil
	}
	for _, kv := range strings.Split(s, "|") {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			return CBData{}, fmt.Errorf("%w: %q", ErrCBMalformed, kv)
		}
		key, val := kv[:eq], kv[eq+1:]
		switch key {
		case "t":
			d.FileType = val
		case "p":
			n, err := strconv.Atoi(val)
			if err != nil {
				return CBData{}, fmt.Errorf("%w: page=%q", ErrCBMalformed, val)
			}
			d.Page = n
		case "f":
			n, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return CBData{}, fmt.Errorf("%w: file=%q", ErrCBMalformed, val)
			}
			d.FileDBID = uint(n)
		case "u":
			n, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return CBData{}, fmt.Errorf("%w: user=%q", ErrCBMalformed, val)
			}
			d.UserID = uint(n)
		case "tg":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return CBData{}, fmt.Errorf("%w: tg=%q", ErrCBMalformed, val)
			}
			d.TelegramID = n
		case "k":
			d.CacheKey = val
		default:
			return CBData{}, fmt.Errorf("%w: unknown key %q", ErrCBMalformed, key)
		}
	}
	return d, nil
}
