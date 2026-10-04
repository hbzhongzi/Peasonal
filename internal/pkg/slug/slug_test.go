package slug

import "testing"

func TestMake(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Hello World", "hello-world"},
		{"Go 并发模式：从入门到放弃", "go-并发模式-从入门到放弃"},
		{"  （空格与标点）  ", "空格与标点"},
		{"a---b", "a-b"},
		{"!!!", "post"},       // 全是标点，退化成兜底值
		{"", "post"},          // 空标题
		{"Rust & Go / 2026", "rust-go-2026"},
	}
	for _, c := range cases {
		if got := Make(c.in); got != c.want {
			t.Errorf("Make(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMake_LengthLimit(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "字"
	}
	got := Make(long)
	if len([]rune(got)) > MaxLen {
		t.Errorf("slug 超长: %d", len([]rune(got)))
	}
	if got[len(got)-1] == '-' {
		t.Errorf("slug 不应以连字符结尾: %q", got)
	}
}

func TestEnsureUnique(t *testing.T) {
	used := map[string]bool{"go": true, "go-2": true}
	got, ok := EnsureUnique("go", func(s string) bool { return used[s] })
	if !ok || got != "go-3" {
		t.Errorf("EnsureUnique = %q, %v; want go-3, true", got, ok)
	}

	got, ok = EnsureUnique("fresh", func(string) bool { return false })
	if !ok || got != "fresh" {
		t.Errorf("未占用时应原样返回，实际 %q, %v", got, ok)
	}

	// 全部被占用时不能死循环
	if _, ok := EnsureUnique("x", func(string) bool { return true }); ok {
		t.Error("全部占用时应返回 false")
	}
}
