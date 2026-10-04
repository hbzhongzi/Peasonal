// Package domain 定义业务实体。
//
// 这一层不依赖 HTTP，也不依赖 SQL —— 它只是「业务语言」。
// service 消费它，transport 把它序列化成 JSON。
package domain

import (
	"time"

	"personal-blog/internal/pkg/markdown"
)

// PostStatus 对应 posts.status。
type PostStatus int8

const (
	StatusDraft     PostStatus = 1
	StatusPublished PostStatus = 2
	StatusArchived  PostStatus = 3
)

// Tag 是文章标签。
type Tag struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// TagWithCount 是带文章数的标签（标签云/标签页用）。
type TagWithCount struct {
	Tag
	PostCount int64 `json:"post_count"`
}

// PostAuthor 是文章作者的公开信息。
//
// 为什么列表也要带作者：站上不止一个作者后，「这是谁写的」不能再靠猜，
// 前台要能把不同作者的文章区分开、并跳到作者自己的博客页（/blog/{public_id}）。
// 刻意不含邮箱 —— 这个结构会出现在匿名可读的公开接口里。
type PostAuthor struct {
	PublicID    string `json:"public_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

// PostListItem 是列表页用的精简结构 —— 不含正文。
type PostListItem struct {
	PublicID       string     `json:"public_id"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	CoverURL       string     `json:"cover_url"`
	ReadingTimeMin int        `json:"reading_time_min"`
	PublishedAt    *time.Time `json:"published_at"`
	Tags           []Tag      `json:"tags"`
	Author         PostAuthor `json:"author"`
}

// Post 是文章详情，含已渲染的正文与目录。
type Post struct {
	PublicID       string             `json:"public_id"`
	Slug           string             `json:"slug"`
	Title          string             `json:"title"`
	Summary        string             `json:"summary"`
	ContentHTML    string             `json:"content_html"`
	TOC            []markdown.TOCItem `json:"toc"`
	CoverURL       string             `json:"cover_url"`
	ReadingTimeMin int                `json:"reading_time_min"`
	CommentEnabled bool               `json:"comment_enabled"`
	PublishedAt    *time.Time         `json:"published_at"`
	Tags           []Tag              `json:"tags"`
	Author         PostAuthor         `json:"author"`
}

// ArchiveMonth 是归档分组（按年月）。
type ArchiveMonth struct {
	Year  int    `json:"year"`
	Month int    `json:"month"`
	Label string `json:"label"`
	Total int64  `json:"total"`
}

// Page 是统一的分页容器。
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// TotalPages 返回总页数。
func (p Page[T]) TotalPages() int {
	if p.PageSize <= 0 || p.Total == 0 {
		return 0
	}
	return int((p.Total + int64(p.PageSize) - 1) / int64(p.PageSize))
}
