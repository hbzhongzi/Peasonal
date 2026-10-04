// Package slug 生成文章 URL 片段。
package slug

import (
	"strings"
	"unicode"
)

// MaxLen 是 slug 的最大长度（按 rune 计）。
//
// 限制长度是因为 slug 会进 URL 和索引，过长既影响可读性也浪费索引空间。
const MaxLen = 96

// Make 把标题转成 slug。
//
// 取向（M1-D2）：**保留中文**。中文标题转拼音后往往不可读（"go-de-bing-fa"），
// 而现代浏览器会自动把 UTF-8 URL 显示为可读中文，搜索引擎也能正确处理。
// 需要短链接时由作者在编辑器里手动覆盖 slug。
func Make(title string) string {
	var b strings.Builder
	pending := false

	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if pending && b.Len() > 0 {
				b.WriteRune('-')
			}
			pending = false
			b.WriteRune(unicode.ToLower(r))
		case r == '-' || r == '_':
			// 连字符同样走 pending：连续多个分隔符只保留一个（a---b → a-b）
			pending = true
		default:
			// 空格、标点、控制字符统一收敛成一个连字符
			pending = true
		}
	}

	s := strings.Trim(b.String(), "-_")
	if s == "" {
		return "post"
	}
	return truncate(s, MaxLen)
}

// truncate 按 rune 截断，并且不留在半截连字符上。
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	cut := max
	for cut > 0 && runes[cut-1] == '-' {
		cut--
	}
	return strings.TrimRight(string(runes[:cut]), "-")
}

// EnsureUnique 在 base 已被占用时追加序号，直到 exists 返回 false。
//
// exists 通常是一次数据库查询；调用方应控制重试次数，这里上限设为 100 防止死循环。
func EnsureUnique(base string, exists func(string) bool) (string, bool) {
	if base == "" {
		base = "post"
	}
	if !exists(base) {
		return base, true
	}
	for i := 2; i <= 100; i++ {
		candidate := base + "-" + itoa(i)
		if !exists(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func itoa(i int) string {
	var b [8]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
