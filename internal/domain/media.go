package domain

import "time"

// MediaItem 是媒体库的元数据（不含文件二进制）。
type MediaItem struct {
	PublicID  string    `json:"public_id"`
	URL       string    `json:"url"`
	Filename  string    `json:"filename"`
	SizeBytes uint64    `json:"size_bytes"`
	MimeType  string    `json:"mime_type"`
	Width     uint32    `json:"width"`
	Height    uint32    `json:"height"`
	CreatedAt time.Time `json:"created_at"`
}
