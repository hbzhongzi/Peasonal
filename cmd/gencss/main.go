// Command gencss 生成代码高亮的 CSS，供前端直接引用。
//
// 为什么要单独生成而不是手写：chroma 的 class 名有几十个（k / nf / s / c / ...），
// 手抄既容易漏也容易错。这里用 chroma 自带的 WriteCSS 导出，
// 亮色写在 `.chroma` 下，暗色写在 `.dark .chroma` 下 ——
// 切主题只是换 class，不需要重新渲染 HTML，也不需要客户端 JS。
//
// 用法：make css
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	// 注意：chroma 内置的亮色样式名是 "github"，**没有** "github-light"。
	// 写错不会报错，styles.Get 会静默返回黑底的 Fallback —— 已经踩过一次。
	lightStyle = "github"
	darkStyle  = "github-dark"
)

func main() {
	out := flag.String("out", "web/src/styles/chroma.css", "输出文件路径")
	flag.Parse()

	f := chromahtml.New(chromahtml.WithClasses(true))

	var light, dark bytes.Buffer
	if err := f.WriteCSS(&light, mustStyle(lightStyle)); err != nil {
		fatal(err)
	}
	if err := f.WriteCSS(&dark, mustStyle(darkStyle)); err != nil {
		fatal(err)
	}

	var b bytes.Buffer
	b.WriteString("/* 自动生成，请勿手工修改 —— 运行 make css 重新生成 */\n")
	b.WriteString("/* 亮色：github-light ／ 暗色：github-dark（挂在 .dark 下） */\n\n")
	b.WriteString("/* ---- 亮色（默认） ---- */\n")
	b.Write(light.Bytes())
	b.WriteString("\n/* ---- 暗色：站点切到 .dark 时生效 ---- */\n")
	b.WriteString(scopeUnderDark(dark.String()))
	b.WriteString("\n/* 代码块容器：圆角、横向滚动、字号 */\n")
	b.WriteString(`.code-block {
  position: relative;
  margin: 1.25em 0;
  border-radius: 8px;
  overflow: hidden;
}

.code-block pre {
  margin: 0;
  padding: 1rem 1.125rem;
  overflow-x: auto;
  font-size: 0.875rem;
  line-height: 1.7;
}
`)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*out, b.Bytes(), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("已生成 %s（亮色 %d 字节 / 暗色 %d 字节）\n", *out, light.Len(), dark.Len())
}

// scopeUnderDark 把 `.chroma ...` 全部改写到 `.dark .chroma ...` 作用域下。
func scopeUnderDark(css string) string {
	return strings.ReplaceAll(css, ".chroma", ".dark .chroma")
}

func mustStyle(name string) *chroma.Style {
	s := styles.Get(name)
	if s == nil {
		fatal(fmt.Errorf("未找到 chroma 样式: %s", name))
	}
	return s
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "gencss: %v\n", err)
	os.Exit(1)
}
