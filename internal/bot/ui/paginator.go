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

// FormatFileLine 将单行 "<序号>. <图标> <文件名> (<文件大小>) - <创建日期>" 渲染为 HTML
// 文件名会被包裹在一个deep link中，该deep link指向 t.me/<botUsername>?start=file_<ID>
// 该方法由列表视图和搜索视图共享
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

// FormatFileList 组装一页文件的列表主体内容
// `header` 是已格式化好的标题行（例如 "📁 我的文件库 (共 12 个文件)\n\n"）
// `botUsername` 用于为每个文件名生成对应的 deep-link 锚点链接
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

// FileTypeFilterRow 返回由“文档/音频/视频/图片/全部”按钮组成的按钮行
// `cbKind` 是应用于每个按钮的 InlineButton.Unique 标识；
// `dataFor(ft)` 用于为给定的文件类型生成编码后的data payload
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

// PaginationRow 返回 "◀ 上一页 / 第X/Y页 / 下一页 ▶" 分页按钮行
// `dataFor(page)` 用于为特定的目标页码生成编码后的数据
// 如果超出边界（即没有前一页或后一页），将自动省略“上一页”或“下一页”按钮
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

// FileNumberButtons 将文件序号按钮进行打包排列，每行最多展示 5 个
// 每个按钮所绑定的数据为经过编码的、包含 FileDBID 的 CBData
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
