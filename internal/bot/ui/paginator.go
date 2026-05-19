package ui

import (
	"fmt"
	"html"
	"strings"

	"tg-drive-bot/internal/constants"
	"tg-drive-bot/internal/model"
	"tg-drive-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

// FormatFileLine renders a single "<N>. <icon> <name> (<size>) - <date>" line as HTML.
// The filename is wrapped in a deep-link anchor pointing to t.me/<botUsername>?start=file_<ID>.
// Shared by list and search views (identical rendering by design).
func FormatFileLine(index int, f *model.File, botUsername string) string {
	icon := util.FileTypeIcon(f.FileType)
	date := f.CreatedAt.Format(constants.DateShort)
	name := f.FileName
	if name == "" {
		name = util.FileTypeName(f.FileType)
	}
	deepLink := fmt.Sprintf("https://t.me/%s?start=file_%s", botUsername, f.FileUniqueID)
	nameLinked := fmt.Sprintf(`<a href="%s">%s</a>`, deepLink, html.EscapeString(name))
	line := fmt.Sprintf("%d. %s %s (%s) - %s\n", index, icon, nameLinked, util.FormatFileSize(f.FileSize), date)
	if f.Caption != "" {
		preview := util.CompactLines(util.TruncateString(html.EscapeString(f.Caption), 50))
		if preview != "" {
			line += fmt.Sprintf("   📝 %s\n\n", preview)
		}
	}
	return line
}

// FormatFileList assembles the list body for a page of files.
// `header` is the already-formatted title line (e.g. "📁 我的文件库 (共 12 个文件)\n\n").
// `botUsername` is used to generate deep-link anchors for each filename.
func FormatFileList(header string, files []model.File, page, pageSize int, botUsername string) string {
	if len(files) == 0 {
		return header
	}
	var sb strings.Builder
	sb.WriteString(header)
	offset := (page - 1) * pageSize
	for i := range files {
		sb.WriteString(FormatFileLine(offset+i+1, &files[i], botUsername))
	}
	return sb.String()
}

// FileTypeFilterRow returns the "文档/音频/视频/图片/全部" row.
// `cbKind` is the InlineButton.Unique used for each button; `dataFor(ft)` produces the
// encoded data payload for a given file type.
func FileTypeFilterRow(markup *tele.ReplyMarkup, cbKind CBKind, dataFor func(ft constants.FileType) string) tele.Row {
	kind := cbKind.String()
	return markup.Row(
		markup.Data("📄 文档", kind, dataFor(constants.FileTypeDocument)),
		markup.Data("🎵 音频", kind, dataFor(constants.FileTypeAudio)),
		markup.Data("🎬 视频", kind, dataFor(constants.FileTypeVideo)),
		markup.Data("🖼 图片", kind, dataFor(constants.FileTypePhoto)),
		markup.Data("全部", kind, dataFor(constants.FileTypeAll)),
	)
}

// PaginationRow returns the "◀ 上一页 / 第X/Y页 / 下一页 ▶" row.
// `dataFor(page)` produces the encoded data for a specific target page.
// `prev`/`next` buttons are omitted if out of bounds.
func PaginationRow(markup *tele.ReplyMarkup, cbKind CBKind, pag *util.Pagination, dataFor func(page int) string) tele.Row {
	kind := cbKind.String()
	var btns []tele.Btn
	if pag.HasPrev() {
		btns = append(btns, markup.Data("◀ 上一页", kind, dataFor(pag.Page-1)))
	}
	btns = append(btns, markup.Data(pag.PageLabel(), CBNoop.String(), "0"))
	if pag.HasNext() {
		btns = append(btns, markup.Data("下一页 ▶", kind, dataFor(pag.Page+1)))
	}
	return markup.Row(btns...)
}

// FileNumberButtons packs file-index buttons into rows of 5.
// Each button's data is the encoded CBData containing FileDBID.
func FileNumberButtons(markup *tele.ReplyMarkup, files []model.File, page, pageSize int) []tele.Row {
	if len(files) == 0 {
		return nil
	}
	var buttons []tele.Btn
	for i := range files {
		num := (page-1)*pageSize + i + 1
		data := Encode(CBData{FileDBID: files[i].ID})
		buttons = append(buttons, markup.Data(fmt.Sprintf("%d", num), CBFile.String(), data))
	}
	var rows []tele.Row
	for i := 0; i < len(buttons); i += 5 {
		end := i + 5
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, markup.Row(buttons[i:end]...))
	}
	return rows
}
