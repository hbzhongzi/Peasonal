package service

import (
	"database/sql"
	"encoding/json"

	"github.com/oklog/ulid/v2"

	"personal-blog/internal/pkg/markdown"
)

// newULID 生成对外标识。ULID 比 UUID 可读、且按时间大致有序。
func newULID() string { return ulid.Make().String() }

func boolToInt8(b bool) int8 {
	if b {
		return 1
	}
	return 0
}

// tocNullString 把目录序列化进可空字符串列（toc_json 用 TEXT 存 JSON）。
func tocNullString(toc []markdown.TOCItem) sql.NullString {
	if len(toc) == 0 {
		return sql.NullString{}
	}
	b, err := json.Marshal(toc)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// readingTime 估算阅读时长（分钟）。中文按 ~400 字/分钟，英文按 ~250 词/分钟，
// 取较大值避免低估；至少 1 分钟。
func readingTime(plain string) uint32 {
	runes := []rune(plain)
	cjk := 0
	words := 1
	inWord := false
	for _, r := range plain {
		if r > 0x2E80 && r < 0x9FFF { // 中日韩统一表意文字区间，覆盖绝大部分中文
			cjk++
		}
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '　':
			inWord = false
		default:
			if !inWord {
				words++
				inWord = true
			}
		}
	}
	_ = runes
	byCJK := cjk / 400
	byWords := words / 250
	mins := byCJK
	if byWords > mins {
		mins = byWords
	}
	if mins < 1 {
		mins = 1
	}
	return uint32(mins)
}
