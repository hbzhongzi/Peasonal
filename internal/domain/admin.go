package domain

import "time"

// AdminUser 是登录/管理用户的基本信息（不含密码哈希）。
type AdminUser struct {
	ID          uint64 `json:"id"`
	PublicID    string `json:"public_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	CanUpload   bool   `json:"can_upload"`
	// CanPublish 决定能否发布/管理文章，由管理员在用户管理页开启。
	CanPublish bool `json:"can_publish"`
	// IsAdmin 决定能否进入纯管理后台（用户/权限管理）。与 CanPublish 解耦：
	// 普通作者只有 CanPublish，看不到也进不了后台。
	IsAdmin bool `json:"is_admin"`
	// 以下三项是「博主的公开身份」，由本人具名修改（PATCH /admin/me），
	// 不再写死在前端常量里 —— 否则换个人登录看到的还是同一个人的简介。
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	Github    string `json:"github"`
}

// SiteProfile 是前台展示的「站长公开资料」。
//
// 只暴露愿意公开的三项 + 昵称，不含邮箱：这个接口是匿名的，
// 谁都能调，多返回一个字段就是多一份泄露面。
type SiteProfile struct {
	PublicID    string `json:"public_id"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Github      string `json:"github"`
}

// MyStats 是「我的视角」发布统计（GET /admin/me/stats 返回）：
// 首页 Hero 与侧边栏个人卡片在登录态下展示当前用户自己的数字，
// 不再永远显示站长的。只统计已发布（status=2），与公开侧口径一致。
type MyStats struct {
	PostTotal  int64 `json:"post_total"`
	TagTotal   int64 `json:"tag_total"`
	MonthTotal int64 `json:"month_total"`
}

// PublicProfile 是任意用户的公开资料（GET /users/{public_id} 返回，用户的博客主页用）。
// 与 SiteProfile 字段一致，但语义不同：那个是「站长」，这个是「任意注册用户」。
// 同样刻意不含邮箱与权限位 —— 匿名接口，多一个字段就多一份泄露面。
type PublicProfile struct {
	PublicID    string `json:"public_id"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Github      string `json:"github"`
}

// AdminPostListItem 是后台列表用的精简结构。
type AdminPostListItem struct {
	PublicID       string     `json:"public_id"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	CoverURL       string     `json:"cover_url"`
	Status         PostStatus `json:"status"`
	CommentEnabled bool       `json:"comment_enabled"`
	ReadingTimeMin int        `json:"reading_time_min"`
	PublishedAt    *time.Time `json:"published_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AdminPostDetail 是后台编辑页用的完整结构（含正文原文与标签）。
type AdminPostDetail struct {
	PublicID       string     `json:"public_id"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	ContentMd      string     `json:"content_md"`
	CoverURL       string     `json:"cover_url"`
	Status         PostStatus `json:"status"`
	CommentEnabled bool       `json:"comment_enabled"`
	PublishedAt    *time.Time `json:"published_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Tags           []string   `json:"tags"`
}
