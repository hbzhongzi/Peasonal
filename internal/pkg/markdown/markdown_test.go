package markdown

import (
	"strings"
	"testing"
)

const sample = "```go\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```"

func TestRender_Highlight_UsesChromaClasses(t *testing.T) {
	r := New(WithCacheSize(0))
	res, err := r.Render("# 标题\n\n" + sample)
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}

	if !strings.Contains(res.HTML, `class="chroma"`) {
		t.Errorf("未输出 chroma 容器，实际: %s", res.HTML)
	}
	if !strings.Contains(res.HTML, `<span class="`) {
		t.Errorf("未输出 class-based 高亮，实际: %s", res.HTML)
	}
	if strings.Contains(res.HTML, "style=") {
		t.Errorf("高亮不应输出内联样式（否则无法用 CSS 切主题），实际: %s", res.HTML)
	}
	if !strings.Contains(res.HTML, `class="code-block"`) {
		t.Errorf("未包 code-block 容器，实际: %s", res.HTML)
	}
	t.Logf("高亮输出: %s", res.HTML)
}

func TestRender_HeadingAnchor_KeepsChinese(t *testing.T) {
	r := New(WithCacheSize(0))
	res, err := r.Render("## Go 并发模式\n\n正文\n\n## Go 并发模式\n")
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}

	if len(res.TOC) != 2 {
		t.Fatalf("TOC 应有 2 项，实际 %d: %+v", len(res.TOC), res.TOC)
	}
	if res.TOC[0].Level != 2 {
		t.Errorf("层级应为 2，实际 %d", res.TOC[0].Level)
	}
	if !strings.Contains(res.TOC[0].ID, "go") {
		t.Errorf("中文标题的 anchor 应保留中文并保留 ascii 部分，实际: %q", res.TOC[0].ID)
	}
	// 重复标题必须去重，否则锚点跳转全指向第一个
	if res.TOC[0].ID == res.TOC[1].ID {
		t.Errorf("重复标题的 anchor 未去重: %q", res.TOC[0].ID)
	}
	t.Logf("TOC: %+v", res.TOC)
}

func TestRender_ExternalLink(t *testing.T) {
	r := New(WithCacheSize(0), WithSiteHost("example.com"))
	res, err := r.Render("[外站](https://other.com/a) 与 [本站](https://example.com/b) 与 [相对](/about)")
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}

	if !strings.Contains(res.HTML, `rel="noopener noreferrer"`) {
		t.Errorf("外链未加 rel，实际: %s", res.HTML)
	}
	if strings.Count(res.HTML, `target="_blank"`) != 1 {
		t.Errorf("只有外链应加 target=_blank，实际出现 %d 次: %s",
			strings.Count(res.HTML, `target="_blank"`), res.HTML)
	}
}

func TestRender_Image(t *testing.T) {
	r := New(WithCacheSize(0), WithImageResolver(fixedResolver{800, 600}))
	res, err := r.Render("![](/uploads/2026/09/a.webp)\n")
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}

	for _, want := range []string{`loading="lazy"`, `width="800"`, `height="600"`, "<figure>"} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("图片处理缺少 %s，实际: %s", want, res.HTML)
		}
	}
}

func TestRender_SanitizesXSS(t *testing.T) {
	r := New(WithCacheSize(0))
	for _, evil := range []string{
		`<script>alert(1)</script>`,
		`<img src=x onerror=alert(1)>`,
		`<a href="javascript:alert(1)">x</a>`,
	} {
		res, err := r.Render(evil)
		if err != nil {
			t.Fatalf("Render 失败: %v", err)
		}
		lower := strings.ToLower(res.HTML)
		if strings.Contains(lower, "onerror") || strings.Contains(lower, "javascript:") || strings.Contains(lower, "<script") {
			t.Errorf("消毒未拦住 %q，输出: %s", evil, res.HTML)
		}
	}
}

func TestRender_GFMTableAndTaskList(t *testing.T) {
	r := New(WithCacheSize(0))
	res, err := r.Render("| a | b |\n| - | - |\n| 1 | 2 |\n\n- [x] 已完成\n- [ ] 未完成\n")
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}
	if !strings.Contains(res.HTML, "<table>") {
		t.Errorf("GFM 表格未渲染: %s", res.HTML)
	}
	if !strings.Contains(res.HTML, `type="checkbox"`) {
		t.Errorf("任务列表未渲染: %s", res.HTML)
	}
}

func TestRender_CacheHitReturnsSameResult(t *testing.T) {
	r := New() // 默认开缓存
	a, err := r.Render(sample)
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}
	b, err := r.Render(sample)
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}
	if a != b {
		t.Errorf("缓存未命中，两次返回了不同对象")
	}
}

func TestExcerpt(t *testing.T) {
	long := strings.Repeat("这是一段很长的正文内容。", 50)
	got := Excerpt(long, 100)
	if len([]rune(got)) > 101 {
		t.Errorf("摘要超长: %d", len([]rune(got)))
	}
	if strings.Contains(got, "<") {
		t.Errorf("摘要不应含标签: %s", got)
	}
	if got == "" {
		t.Error("摘要为空")
	}
	t.Logf("摘要: %s", got)

	short := "很短"
	if Excerpt(short, 100) != short {
		t.Errorf("短文本应原样返回，实际: %q", Excerpt(short, 100))
	}
	if Excerpt(short, 0) != "" {
		t.Errorf("limit<=0 应返回空串")
	}
}

// fixedResolver 总是返回固定尺寸，用于测试。
type fixedResolver struct{ w, h int }

func (f fixedResolver) Resolve(string) (int, int, bool) { return f.w, f.h, true }

// 文章主标题（h1）已经在页面顶部单独渲染，再进目录或摘要就是重复一遍。
// 这两条曾经是真实缺陷：列表页摘要一开头就是标题，目录第一项也是标题。
func TestRender_H1ExcludedFromTOCAndExcerpt(t *testing.T) {
	r := New(WithCacheSize(0))
	src := "# 文章主标题\n\n这是真正的正文开头。\n\n## 第一个章节\n\n章节内容。\n"
	res, err := r.Render(src)
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}

	if len(res.TOC) != 1 {
		t.Fatalf("TOC 只应有 h2 那一项，实际 %d: %+v", len(res.TOC), res.TOC)
	}
	if res.TOC[0].Text != "第一个章节" {
		t.Errorf("TOC 首项应为第一个 h2，实际: %q", res.TOC[0].Text)
	}

	// h1 仍然要带 id，文内锚点可能指向它
	if !strings.Contains(res.HTML, `<h1 id="`) {
		t.Errorf("h1 应保留 anchor id，实际: %s", res.HTML)
	}

	if strings.Contains(res.PlainText, "文章主标题") {
		t.Errorf("PlainText 不应包含 h1 标题，实际: %q", res.PlainText)
	}
	if !strings.HasPrefix(res.PlainText, "这是真正的正文开头") {
		t.Errorf("PlainText 应从正文开始，实际: %q", res.PlainText)
	}
}

// 行内代码要保留（否则句子残缺），块级代码要跳过（否则摘要混进一大段代码）。
// 这两条方向相反，容易改坏其中一个，所以放在一起测。
func TestRender_PlainTextCodeHandling(t *testing.T) {
	r := New(WithCacheSize(0))
	src := "使用 `errgroup.WithContext` 可以取消。\n\n```go\nfunc leakedCode() {}\n```\n\n结尾。"
	res, err := r.Render(src)
	if err != nil {
		t.Fatalf("Render 失败: %v", err)
	}

	if !strings.Contains(res.PlainText, "errgroup.WithContext") {
		t.Errorf("行内代码应保留在 PlainText 中，实际: %q", res.PlainText)
	}
	if strings.Contains(res.PlainText, "leakedCode") {
		t.Errorf("块级代码不应出现在 PlainText 中，实际: %q", res.PlainText)
	}
}
