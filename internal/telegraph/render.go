package telegraph

import (
	"fmt"
	"strings"
)

// RenderMarkdown converts a Page into a Markdown document. Media nodes (img,
// video, iframe) are kept as links — their src attributes are preserved
// verbatim so the user can still reach the original media.
func RenderMarkdown(page *Page) string {
	var b strings.Builder

	if page.Title != "" {
		fmt.Fprintf(&b, "# %s\n\n", page.Title)
	}
	if page.AuthorName != "" {
		fmt.Fprintf(&b, "> Author: %s\n\n", page.AuthorName)
	}
	if page.Path != "" {
		fmt.Fprintf(&b, "> Source: https://telegra.ph/%s\n\n", page.Path)
	}

	r := &renderer{}
	for _, n := range page.Content {
		r.renderBlock(&b, n)
	}

	out := strings.TrimRight(b.String(), "\n") + "\n"
	return out
}

type renderer struct {
	listDepth int
	orderedAt []bool // parallel stack: true if current list is ordered
	liIndex   []int  // parallel stack: running index for ordered lists
}

// renderBlock handles block-level (standalone) nodes that produce their own
// paragraph breaks.
func (r *renderer) renderBlock(b *strings.Builder, n Node) {
	if n.IsText() {
		txt := strings.TrimSpace(n.Text)
		if txt == "" {
			return
		}
		b.WriteString(txt)
		b.WriteString("\n\n")
		return
	}

	switch n.Tag {
	case "p":
		inline := r.renderInline(n.Children)
		if s := strings.TrimSpace(inline); s != "" {
			b.WriteString(s)
			b.WriteString("\n\n")
		}
	case "h3":
		b.WriteString("## ")
		b.WriteString(strings.TrimSpace(r.renderInline(n.Children)))
		b.WriteString("\n\n")
	case "h4":
		b.WriteString("### ")
		b.WriteString(strings.TrimSpace(r.renderInline(n.Children)))
		b.WriteString("\n\n")
	case "blockquote":
		r.renderBlockquote(b, n.Children)
	case "aside":
		r.renderBlockquote(b, n.Children)
	case "ul":
		r.renderList(b, n.Children, false)
	case "ol":
		r.renderList(b, n.Children, true)
	case "pre":
		r.renderPre(b, n.Children)
	case "hr":
		b.WriteString("---\n\n")
	case "figure":
		r.renderFigure(b, n.Children)
	case "img":
		writeImage(b, n.Attrs)
		b.WriteString("\n\n")
	case "video":
		writeMediaLink(b, "Video", n.Attrs)
		b.WriteString("\n\n")
	case "iframe":
		writeMediaLink(b, "Embedded", n.Attrs)
		b.WriteString("\n\n")
	case "br":
		b.WriteString("\n")
	default:
		// Unknown block tag — flatten its children as inline text.
		inline := r.renderInline(n.Children)
		if s := strings.TrimSpace(inline); s != "" {
			b.WriteString(s)
			b.WriteString("\n\n")
		}
	}
}

// renderInline handles nodes that appear inside a paragraph or heading.
func (r *renderer) renderInline(nodes []Node) string {
	var b strings.Builder
	for _, n := range nodes {
		if n.IsText() {
			b.WriteString(n.Text)
			continue
		}
		switch n.Tag {
		case "a":
			href := n.Attrs["href"]
			inner := r.renderInline(n.Children)
			if href == "" {
				b.WriteString(inner)
				continue
			}
			fmt.Fprintf(&b, "[%s](%s)", inner, href)
		case "strong", "b":
			fmt.Fprintf(&b, "**%s**", r.renderInline(n.Children))
		case "em", "i":
			fmt.Fprintf(&b, "*%s*", r.renderInline(n.Children))
		case "u":
			// Markdown has no underline; use emphasis as the closest fit.
			fmt.Fprintf(&b, "*%s*", r.renderInline(n.Children))
		case "s", "strike", "del":
			fmt.Fprintf(&b, "~~%s~~", r.renderInline(n.Children))
		case "code":
			fmt.Fprintf(&b, "`%s`", r.renderInline(n.Children))
		case "br":
			b.WriteString("  \n")
		case "img":
			writeImage(&b, n.Attrs)
		case "video":
			writeMediaLink(&b, "Video", n.Attrs)
		case "iframe":
			writeMediaLink(&b, "Embedded", n.Attrs)
		default:
			b.WriteString(r.renderInline(n.Children))
		}
	}
	return b.String()
}

func (r *renderer) renderBlockquote(b *strings.Builder, children []Node) {
	var inner strings.Builder
	for _, c := range children {
		r.renderBlock(&inner, c)
	}
	text := strings.TrimRight(inner.String(), "\n")
	if text == "" {
		return
	}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			b.WriteString(">\n")
			continue
		}
		fmt.Fprintf(b, "> %s\n", line)
	}
	b.WriteString("\n")
}

func (r *renderer) renderList(b *strings.Builder, items []Node, ordered bool) {
	r.listDepth++
	r.orderedAt = append(r.orderedAt, ordered)
	r.liIndex = append(r.liIndex, 0)
	defer func() {
		r.listDepth--
		r.orderedAt = r.orderedAt[:len(r.orderedAt)-1]
		r.liIndex = r.liIndex[:len(r.liIndex)-1]
	}()

	for _, item := range items {
		if item.IsText() || item.Tag != "li" {
			continue
		}
		r.liIndex[len(r.liIndex)-1]++
		indent := strings.Repeat("  ", r.listDepth-1)

		var marker string
		if ordered {
			marker = fmt.Sprintf("%d. ", r.liIndex[len(r.liIndex)-1])
		} else {
			marker = "- "
		}

		inline := strings.TrimSpace(r.renderInline(item.Children))
		// Handle nested blocks (nested lists) by rendering them separately below the li line.
		var nested strings.Builder
		for _, c := range item.Children {
			if !c.IsText() && (c.Tag == "ul" || c.Tag == "ol") {
				r.renderBlock(&nested, c)
			}
		}

		fmt.Fprintf(b, "%s%s%s\n", indent, marker, inline)
		if nested.Len() > 0 {
			for _, line := range strings.Split(strings.TrimRight(nested.String(), "\n"), "\n") {
				fmt.Fprintf(b, "%s  %s\n", indent, line)
			}
		}
	}
	b.WriteString("\n")
}

func (r *renderer) renderPre(b *strings.Builder, children []Node) {
	b.WriteString("```\n")
	for _, c := range children {
		if c.IsText() {
			b.WriteString(c.Text)
			continue
		}
		// Unwrap inner <code> / other inline wrappers.
		b.WriteString(r.renderInline(c.Children))
	}
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString("```\n\n")
}

func (r *renderer) renderFigure(b *strings.Builder, children []Node) {
	// Typical shape: figure > [img|video|iframe, figcaption]
	var caption string
	for _, c := range children {
		if !c.IsText() && c.Tag == "figcaption" {
			caption = strings.TrimSpace(r.renderInline(c.Children))
		}
	}
	for _, c := range children {
		if c.IsText() {
			continue
		}
		switch c.Tag {
		case "img":
			writeImageWithAlt(b, c.Attrs, caption)
			b.WriteString("\n")
		case "video":
			writeMediaLink(b, "Video", c.Attrs)
			b.WriteString("\n")
		case "iframe":
			writeMediaLink(b, "Embedded", c.Attrs)
			b.WriteString("\n")
		}
	}
	if caption != "" {
		fmt.Fprintf(b, "*%s*\n", caption)
	}
	b.WriteString("\n")
}

func writeImage(b *strings.Builder, attrs map[string]string) {
	writeImageWithAlt(b, attrs, "")
}

func writeImageWithAlt(b *strings.Builder, attrs map[string]string, alt string) {
	src := absoluteURL(attrs["src"])
	if src == "" {
		return
	}
	fmt.Fprintf(b, "![%s](%s)", alt, src)
}

func writeMediaLink(b *strings.Builder, label string, attrs map[string]string) {
	src := absoluteURL(attrs["src"])
	if src == "" {
		return
	}
	fmt.Fprintf(b, "[%s](%s)", label, src)
}

// absoluteURL rewrites telegra.ph-relative paths (e.g. "/file/xxx.jpg") into
// absolute URLs so links remain usable after the file leaves Telegram.
func absoluteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return "https://" + defaultHost + raw
	}
	return raw
}
