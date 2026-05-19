// Package ui encapsulates Telegram UI concerns: callback data encoding,
// keyboard construction, and paginated list rendering.
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

// CBData holds the decoded payload carried in InlineButton.Data.
// Fields are optional; producers set only what they need, consumers read what they expect.
type CBData struct {
	FileType   string // "all", "document", ... used by list/search
	Page       int    // pagination target
	FileDBID   uint   // file row id (数据库主键,与 Telegram bot file_id 是两回事)
	UserID     uint   // user row id (admin flows)
	TelegramID int64  // raw telegram user id (admin delete)
	CacheKey   string // search-context cache key
}

// Encode serializes CBData into the "k=v|k=v" format. Empty fields are omitted.
// Always fits within telebot's 64-byte callback_data budget for realistic inputs.
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

// Decode parses "k=v|k=v" into CBData. An empty string yields a zero-value CBData and no error,
// so consumers can treat both "absent" and "present but empty" identically.
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
