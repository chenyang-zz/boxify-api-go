package webcrawl

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/boxify/api-go/internal/core/valuex"
	"golang.org/x/net/html"
)

const defaultTitleMaxRunes = 200

// HTMLExtractor 从 HTML 中抽取标题和可读正文。
type HTMLExtractor struct {
	TitleMaxRunes int
}

// ExtractorOption 修改 HTMLExtractor 的长期配置。
type ExtractorOption func(*HTMLExtractor)

// NewHTMLExtractor 创建 HTML 正文提取器。
//
// 默认会把标题裁剪到 200 个 rune，正文不做长度裁剪。
func NewHTMLExtractor(opts ...ExtractorOption) *HTMLExtractor {
	extractor := &HTMLExtractor{
		TitleMaxRunes: defaultTitleMaxRunes,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(extractor)
		}
	}
	return extractor
}

// WithTitleMaxRunes 设置标题最大 rune 数。
//
// maxRunes 小于等于 0 时保留默认值。
func WithTitleMaxRunes(maxRunes int) ExtractorOption {
	return func(extractor *HTMLExtractor) {
		if maxRunes > 0 {
			extractor.TitleMaxRunes = maxRunes
		}
	}
}

// Extract 从 HTML 页面中提取标题和正文。
//
// 正文为空时返回错误；标题为空时使用页面 URL 作为标题。
func (e *HTMLExtractor) Extract(ctx context.Context, page Page) (*Output, error) {
	doc, err := html.Parse(bytes.NewReader(page.HTML))
	if err != nil {
		return nil, err
	}
	title := firstTitle(doc)
	content := readableText(doc)
	if content == "" {
		return nil, errors.New("web page content is empty")
	}
	if title == "" {
		title = page.URL
	}
	return &Output{Title: valuex.TruncateRunesWithSuffix(title, e.TitleMaxRunes, "..."), Content: content, URL: page.URL}, nil
}

// firstTitle 提取网页第一个 title 文本。
func firstTitle(root *html.Node) string {
	var title string
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inTitle bool) {
		if n == nil || title != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "title" {
			inTitle = true
		}
		if inTitle && n.Type == html.TextNode {
			title = strings.TrimSpace(n.Data)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child, inTitle)
		}
	}
	walk(root, false)
	return normalizeSpace(title)
}

// readableText 提取网页可读文本。
func readableText(root *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "title") {
			return
		}
		if n.Type == html.TextNode {
			if text := strings.TrimSpace(n.Data); text != "" {
				parts = append(parts, text)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return normalizeSpace(strings.Join(parts, " "))
}

// normalizeSpace 规范化空白字符。
func normalizeSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
