package markdown

import (
	"io"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
)

// chromaHTMLFormatter 是 chroma 的 HTML 输出器。
type chromaHTMLFormatter = chromahtml.Formatter

// newChromaFormatter 构造「class-based」输出器。
//
// 关键点：WithClasses(true) 让 chroma 输出 `<span class="k">` 而不是内联 style。
// 这样明暗两套主题只是两份 CSS（见 cmd/gencss），切主题不用重新渲染 HTML，
// 也不需要任何客户端 JS —— 这是 M1-D1 决策的核心收益。
func newChromaFormatter() *chromaHTMLFormatter {
	return chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.WithPreWrapper(preWrapper{}),
		chromahtml.TabWidth(4),
	)
}

// preWrapper 给代码块包一层 .code-block，方便前端定位并挂复制按钮。
//
// 注意：如果包了这层容器，CSS 选择器就要写成 `.code-block .chroma`，
// 生成样式时用的前缀仍是 `.chroma`，二者不冲突。
type preWrapper struct{}

func (preWrapper) Start(_ bool, _ string) string {
	return `<div class="code-block"><pre class="chroma"><code>`
}

func (preWrapper) End(_ bool) string {
	return `</code></pre></div>`
}

// chromaEscaped 是不高亮时的兜底：转义后包成普通代码块。
func chromaEscaped(w io.Writer, code string) {
	_, _ = io.WriteString(w, `<div class="code-block"><pre class="chroma"><code>`)
	_, _ = io.WriteString(w, escapeHTML(code))
	_, _ = io.WriteString(w, `</code></pre></div>`)
}

func escapeHTML(s string) string {
	var b []byte
	last := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			b = append(b, s[last:i]...)
			b = append(b, '&', 'l', 't', ';')
			last = i + 1
		case '>':
			b = append(b, s[last:i]...)
			b = append(b, '&', 'g', 't', ';')
			last = i + 1
		case '&':
			b = append(b, s[last:i]...)
			b = append(b, '&', 'a', 'm', 'p', ';')
			last = i + 1
		}
	}
	if last == 0 {
		return s
	}
	b = append(b, s[last:]...)
	return string(b)
}

// 确保 chroma 包被使用（formatter 的 Format 签名依赖它）
var _ = chroma.TokenType(0)
