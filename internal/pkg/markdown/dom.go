package markdown

import (
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// postProcess 在消毒后的 HTML 上做结构化补充。
//
// 这里只做四件事，且全部是「添加我们自己构造的属性」，
// 不引入任何来自 Markdown 的内容 —— 因此放在消毒之后是安全的。
//  1. 给 h1-h6 生成 anchor id，并抽取目录
//  2. 给外链补 target=_blank rel="noopener noreferrer"
//  3. 给图片补 loading/decoding 与已知宽高（防 CLS，§6.4）
//  4. 把「只含一张图片的段落」升级为 figure
func postProcess(in, siteHost string, resolver ImageResolver) (out string, toc []TOCItem, plain string, err error) {
	root, err := html.Parse(strings.NewReader(in))
	if err != nil {
		return "", nil, "", err
	}
	body := findBody(root)
	if body == nil {
		return "", nil, "", nil
	}

	var text strings.Builder
	seen := map[string]int{}

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case isHeading(n.Data):
				t := strings.TrimSpace(textOf(n))
				if t != "" {
					id := uniqueID(headingID(t), seen)
					setAttr(n, html.Attribute{Key: "id", Val: id})
					// h1 不进目录：它是文章主标题，已由 title 字段在页面顶部单独渲染。
					// 把它放进 TOC 只会让目录第一项和标题重复。id 仍然加上，供文内锚点使用。
					if level := headingLevel(n.Data); level >= 2 {
						toc = append(toc, TOCItem{ID: id, Text: t, Level: level})
					}
				}
			case n.Data == "a":
				if isExternal(attrVal(n, "href"), siteHost) {
					setAttr(n,
						html.Attribute{Key: "target", Val: "_blank"},
						html.Attribute{Key: "rel", Val: "noopener noreferrer"},
					)
				}
			case n.Data == "img":
				decorateImage(n, resolver)
			case n.Data == "p" && onlyChildIsImage(n):
				n.Data = "figure"
				n.Attr = nil
			}
		}

		if n.Type == html.TextNode && !shouldSkipText(n) {
			text.WriteString(n.Data)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(body)

	var buf strings.Builder
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&buf, c); err != nil {
			return "", nil, "", err
		}
	}

	return buf.String(), toc, collapseSpace(text.String()), nil
}

func findBody(root *html.Node) *html.Node {
	var found *html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "body" {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return found
}

func isHeading(tag string) bool {
	return len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6'
}

func headingLevel(tag string) int { return int(tag[1] - '0') }

func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOf(c))
	}
	return b.String()
}

// shouldSkipText 判断这段文本是否不该计入 PlainText。
//
// 跳过三类：
//   - script / style：非正文
//   - pre（块级代码）：不加进来是因为摘要会长成「……的写法是 func fetchAll(ctx context.」，
//     一大段代码混在摘要里既难看也不达意。代码内容依然能被搜到 —— 搜索走 content_md 原文。
//   - h1（主标题）：否则摘要会以标题开头，和列表页上方的标题重复一遍。
//
// 注意**不跳过**行内 code：它是句子的组成部分，跳过会让摘要断成
// 「最朴素的写法是用 加一个 变量」这种残缺句。
func shouldSkipText(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type != html.ElementNode {
			continue
		}
		switch p.Data {
		case "script", "style", "pre", "h1":
			return true
		}
	}
	return false
}

func setAttr(n *html.Node, attrs ...html.Attribute) {
	for _, a := range attrs {
		replaced := false
		for i := range n.Attr {
			if strings.EqualFold(n.Attr[i].Key, a.Key) {
				n.Attr[i] = a
				replaced = true
				break
			}
		}
		if !replaced {
			n.Attr = append(n.Attr, a)
		}
	}
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func onlyChildIsImage(n *html.Node) bool {
	if n.FirstChild == nil || n.FirstChild.NextSibling != nil {
		return false
	}
	c := n.FirstChild
	if c.Type != html.ElementNode {
		return false
	}
	return c.Data == "img"
}

func decorateImage(n *html.Node, resolver ImageResolver) {
	setAttr(n,
		html.Attribute{Key: "loading", Val: "lazy"},
		html.Attribute{Key: "decoding", Val: "async"},
	)
	if resolver == nil {
		return
	}
	w, h, ok := resolver.Resolve(attrVal(n, "src"))
	if !ok || w <= 0 || h <= 0 {
		return
	}
	setAttr(n,
		html.Attribute{Key: "width", Val: itoa(w)},
		html.Attribute{Key: "height", Val: itoa(h)},
	)
}

func isExternal(href, siteHost string) bool {
	if href == "" {
		return false
	}
	u, err := url.Parse(href)
	if err != nil || u.Host == "" {
		return false // 相对链接或纯锚点，不是外链
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return false
	}
	if siteHost == "" {
		return true
	}
	host := u.Hostname()
	return !strings.EqualFold(host, siteHost) && !strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(siteHost))
}

// uniqueID 保证同一篇文档内 anchor 不重复。
func uniqueID(base string, seen map[string]int) string {
	seen[base]++
	if seen[base] == 1 {
		return base
	}
	id := base + "-" + itoa(seen[base])
	for seen[id] > 0 {
		seen[base]++
		id = base + "-" + itoa(seen[base])
	}
	seen[id] = 1
	return id
}

// headingID 把标题文本转成 URL 友好的 anchor。
//
// 与文章 slug 不同：标题 anchor **保留中文**（M1-D2 的同一取向）——
// 中文 anchor 在现代浏览器与搜索引擎里都能正常工作，
// 强行转拼音反而让「标题 → 锚点」的对应关系变得不可预测。
func headingID(text string) string {
	var b strings.Builder
	pending := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if pending && b.Len() > 0 {
				b.WriteRune('-')
			}
			pending = false
			b.WriteRune(unicode.ToLower(r))
		case r == '-' || r == '_':
			// 连字符同样走 pending：连续多个分隔符只保留一个
			pending = true
		default:
			pending = true
		}
	}
	id := strings.Trim(b.String(), "-_")
	if id == "" {
		return "section"
	}
	return id
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
