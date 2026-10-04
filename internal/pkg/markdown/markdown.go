// Package markdown 把 Markdown 原文渲染成可直接嵌入页面的 HTML。
//
// 设计要点（对应技术方案 §7.1）：
//   - 渲染发生在**写路径**（保存/发布时），结果落库，读路径零渲染开销。
//   - 渲染链路：goldmark → chroma 高亮 → bluemonday 消毒 → DOM 后处理。
//     消毒必须在后处理**之前**：后处理只添加我们自己构造的属性，
//     而所有来自 Markdown 的内容都要先过白名单。
//   - 结果可缓存（进程内 LRU，key 为原文的 sha256）。
package markdown

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/microcosm-cc/bluemonday"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// ImageResolver 为渲染器提供图片的真实宽高，用于输出 width/height 防 CLS（§6.4）。
// 返回 ok=false 时只输出惰性加载属性，不输出尺寸。
type ImageResolver interface {
	Resolve(src string) (width, height int, ok bool)
}

// TOCItem 是目录项。Level 对应 h1-h6。
type TOCItem struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Level int    `json:"level"`
}

// Result 是一次渲染的完整产物。
//
// 返回值会被 LRU 缓存并可能被多个 goroutine 并发读取，调用方**不得修改**其中任何字段。
type Result struct {
	HTML      string    `json:"html"`
	TOC       []TOCItem `json:"toc"`
	PlainText string    `json:"plain_text"`
}

// Renderer 是并发安全的。
type Renderer struct {
	md       goldmark.Markdown
	policy   *bluemonday.Policy
	cache    *lru.Cache[string, *Result]
	resolver ImageResolver
	siteHost string
}

// Option 配置 Renderer。
type Option func(*Renderer)

// WithImageResolver 注入图片尺寸解析器（通常查 media 表）。不传则不输出宽高。
func WithImageResolver(r ImageResolver) Option {
	return func(rd *Renderer) { rd.resolver = r }
}

// WithSiteHost 设置本站域名，用于判断外链。留空则所有 http(s) 链接都按外链处理。
func WithSiteHost(host string) Option {
	return func(r *Renderer) { r.siteHost = host }
}

// WithCacheSize 设置渲染缓存容量，<=0 表示关闭缓存。默认 256。
func WithCacheSize(n int) Option {
	return func(r *Renderer) {
		if n <= 0 {
			r.cache = nil
			return
		}
		c, err := lru.New[string, *Result](n)
		if err != nil {
			panic(fmt.Sprintf("markdown: 创建渲染缓存失败: %v", err))
		}
		r.cache = c
	}
}

// New 构造渲染器。
func New(opts ...Option) *Renderer {
	cache, _ := lru.New[string, *Result](256)

	r := &Renderer{
		policy: newPolicy(),
		cache:  cache,
	}

	for _, opt := range opts {
		opt(r)
	}

	// goldmark 的渲染器注册是「优先级低的后注册、覆盖高优先级」，
	// 内置的 html renderer 优先级为 1000，所以这里用 100 才能覆盖代码块渲染。
	r.md = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(), // 允许原始 HTML，由 bluemonday 兜底消毒
		),
	)
	r.md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(newCodeBlockRenderer(), 100),
	))

	return r
}

// Render 渲染 Markdown 原文。
func (r *Renderer) Render(src string) (*Result, error) {
	key := sha256.Sum256([]byte(src))
	if r.cache != nil {
		if cached, ok := r.cache.Get(hex.EncodeToString(key[:])); ok {
			return cached, nil
		}
	}

	res, err := r.render(src)
	if err != nil {
		return nil, err
	}

	if r.cache != nil {
		r.cache.Add(hex.EncodeToString(key[:]), res)
	}
	return res, nil
}

func (r *Renderer) render(src string) (*Result, error) {
	var buf bytes.Buffer
	if err := r.md.Convert([]byte(src), &buf); err != nil {
		return nil, fmt.Errorf("markdown: goldmark 渲染失败: %w", err)
	}

	// 1) 消毒：所有来自 Markdown 的内容必须先过白名单
	sanitized := r.policy.Sanitize(buf.String())

	// 2) 后处理：加 anchor / 外链属性 / 图片属性，并抽取 TOC 与纯文本
	htmlOut, toc, plain, err := postProcess(sanitized, r.siteHost, r.resolver)
	if err != nil {
		return nil, fmt.Errorf("markdown: 后处理失败: %w", err)
	}

	return &Result{HTML: htmlOut, TOC: toc, PlainText: plain}, nil
}

// newPolicy 构造消毒剂策略。
//
// 即便只有管理员可写也必须有消毒（§7.1）：管理员账号一旦失守，
// 这是最后一道、也是唯一一道能拦住存储型 XSS 的闸。
func newPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	// chroma 的 class-based 输出依赖这些元素上的 class
	p.AllowAttrs("class").OnElements("pre", "code", "span", "div")
	// 任务列表（GFM）会生成 disabled/checked 的 checkbox
	p.AllowAttrs("checked", "disabled", "type").OnElements("input")
	// 允许 figure/figcaption：图片会被包一层（见 postProcess）
	p.AllowElements("figure", "figcaption")
	// 允许站内相对链接与锚点
	p.AllowURLSchemes("http", "https", "mailto")

	return p
}

// --- 代码块高亮（chroma）---

type codeBlockRenderer struct {
	formatter *chromaHTMLFormatter
	style     *chroma.Style
	mu        sync.Mutex
}

func newCodeBlockRenderer() *codeBlockRenderer {
	return &codeBlockRenderer{
		formatter: newChromaFormatter(),
		// class-based 输出不使用内联样式，style 仅用于兜底判断
		style: styles.Fallback,
	}
}

// RegisterFuncs 覆盖缩进代码块与围栏代码块的渲染。
func (r *codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFenced)
	reg.Register(ast.KindCodeBlock, r.renderIndented)
}

func (r *codeBlockRenderer) renderFenced(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*ast.FencedCodeBlock)
	lang := strings.TrimSpace(string(node.Language(source)))
	r.write(w, lang, linesText(node, source))
	return ast.WalkSkipChildren, nil
}

func (r *codeBlockRenderer) renderIndented(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	r.write(w, "", linesText(n, source))
	return ast.WalkSkipChildren, nil
}

func (r *codeBlockRenderer) write(w util.BufWriter, lang, code string) {
	lexer := pickLexer(lang, code)
	if lexer == nil {
		chromaEscaped(w, code)
		return
	}

	// chroma 的 Format 不是并发安全的（内部复用状态），这里串行化
	r.mu.Lock()
	defer r.mu.Unlock()

	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		chromaEscaped(w, code)
		return
	}
	if err := r.formatter.Format(w, r.style, it); err != nil {
		chromaEscaped(w, code)
		return
	}
}

func pickLexer(lang, code string) chroma.Lexer {
	if lang != "" {
		if l := lexers.Get(lang); l != nil {
			return l
		}
	}
	if l := lexers.Analyse(code); l != nil {
		return l
	}
	// 无语言标注且无法猜测时，不做高亮，直接输出纯文本（避免瞎猜导致配色错乱）
	return nil
}

func linesText(n ast.Node, source []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return b.String()
}

// Excerpt 从渲染结果中生成摘要：去掉标签后按字符截断，尽量落在标点或空白边界上。
//
// 直接截断 HTML 字符串会把标签拦腰砍断，所以这里基于纯文本做。
func Excerpt(plain string, limit int) string {
	if limit <= 0 || plain == "" {
		return ""
	}
	runes := []rune(strings.TrimSpace(plain))
	if len(runes) <= limit {
		return string(runes)
	}
	cut := limit
	// 在 [limit*0.6, limit] 区间内从后往前找断句标点，找不到就硬截
	low := limit * 6 / 10
	for i := limit; i >= low && i > 0; i-- {
		switch runes[i] {
		case '。', '！', '？', '；', '．', '.', '!', '?', ';', '\n':
			cut = i + 1
			return strings.TrimSpace(string(runes[:cut]))
		}
	}
	for i := limit; i >= low && i > 0; i-- {
		if runes[i] == ' ' || runes[i] == '　' {
			cut = i
			return strings.TrimSpace(string(runes[:cut])) + "…"
		}
	}
	return strings.TrimSpace(string(runes[:cut])) + "…"
}
