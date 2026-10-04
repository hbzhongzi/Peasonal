package domain

import "time"

// Comment 是前台展示的公开评论（绝不含 email）。
type Comment struct {
	PublicID   string    `json:"public_id"`
	AuthorName string    `json:"author_name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// AdminComment 是后台管理用的评论（含状态）。
type AdminComment struct {
	PublicID   string    `json:"public_id"`
	AuthorName string    `json:"author_name"`
	Content    string    `json:"content"`
	Status     int8      `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// CommentInput 是提交评论的输入。
//
// Honeypot 是蜜罐字段：正常用户看不到、不会填；机器人会填。
// 填了就静默丢弃并返回成功假象（见 CommentService.Create）。
type CommentInput struct {
	AuthorName  string
	AuthorEmail string
	Content     string
	Honeypot    string
}
