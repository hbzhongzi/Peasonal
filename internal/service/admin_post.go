package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"personal-blog/internal/domain"
	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/pkg/markdown"
	"personal-blog/internal/pkg/slug"
	"personal-blog/internal/repository"
)

// AdminPostService 提供文章的写路径（后台 CRUD + 状态切换 + 标签）。
//
// 渲染发生在保存时（§7.1）：标题/正文/标签变更都要重渲染 content_html 与 toc_json，
// 保证二者与 content_md 永远一致。
// 数据归属：所有写操作都要求 Actor 是作者本人或站点管理员（见 Actor.CanTouch），
// 否则返回 403。少了这层校验，任何能进工作台的人都能改删别人的草稿。
type AdminPostService struct {
	q  *repository.Queries
	md *markdown.Renderer
}

func NewAdminPostService(q *repository.Queries, md *markdown.Renderer) *AdminPostService {
	return &AdminPostService{q: q, md: md}
}

// PostUpsertInput 是新建/更新共用的输入。
type PostUpsertInput struct {
	Title          string
	Slug           string // 可选，留空按标题自动生成
	Summary        string // 可选，留空自动截取
	ContentMd      string
	CoverURL       string
	CommentEnabled bool
	Tags           []string
	Publish        bool // true=已发布，false=草稿
}

// Create 新建并（可选）立即发布。
func (s *AdminPostService) Create(ctx context.Context, actor Actor, in PostUpsertInput) (*domain.AdminPostDetail, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.ContentMd = strings.TrimSpace(in.ContentMd)
	if in.Title == "" {
		return nil, apierr.InvalidArgument("标题不能为空")
	}
	if in.ContentMd == "" {
		return nil, apierr.InvalidArgument("正文不能为空")
	}

	res, err := s.md.Render(in.ContentMd)
	if err != nil {
		return nil, apierr.InvalidArgument("Markdown 渲染失败：" + err.Error())
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = markdown.Excerpt(res.PlainText, 200)
	}
	slug, err := s.uniqueSlug(ctx, strings.TrimSpace(in.Slug), in.Title)
	if err != nil {
		return nil, err
	}

	status := domain.StatusDraft
	if in.Publish {
		status = domain.StatusPublished
	}
	now := time.Now().UTC()
	pubAt := sql.NullTime{}
	if in.Publish {
		pubAt = nullTime(now)
	}
	pid := newULID()

	id, err := s.q.CreatePost(ctx, repository.CreatePostParams{
		PublicID:       pid,
		Slug:           slug,
		Title:          in.Title,
		Summary:        summary,
		ContentMd:      in.ContentMd,
		ContentHtml:    res.HTML,
		TocJson:        tocNullString(res.TOC),
		CoverUrl:       strings.TrimSpace(in.CoverURL),
		Status:         int8(status),
		CommentEnabled: boolToInt8(in.CommentEnabled),
		ReadingTimeMin: readingTime(res.PlainText),
		PublishedAt:    pubAt,
		CreatedAt:      now,
		UpdatedAt:      now,
		AuthorID:       actor.UserID,
	})
	if err != nil {
		return nil, fmt.Errorf("插入文章: %w", err)
	}
	if err := s.linkTags(ctx, uint64(id), in.Tags); err != nil {
		return nil, err
	}
	return s.GetForEdit(ctx, actor, pid)
}

// Update 全量更新一篇已存在文章的字段、正文与标签。
func (s *AdminPostService) Update(ctx context.Context, actor Actor, publicID string, in PostUpsertInput) (*domain.AdminPostDetail, error) {
	existing, err := s.q.GetPostForEdit(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("查询待更新文章: %w", err)
	}
	if !actor.CanTouch(existing.AuthorID) {
		return nil, ErrNotOwner
	}

	in.Title = strings.TrimSpace(in.Title)
	in.ContentMd = strings.TrimSpace(in.ContentMd)
	if in.Title == "" {
		return nil, apierr.InvalidArgument("标题不能为空")
	}
	if in.ContentMd == "" {
		return nil, apierr.InvalidArgument("正文不能为空")
	}

	res, err := s.md.Render(in.ContentMd)
	if err != nil {
		return nil, apierr.InvalidArgument("Markdown 渲染失败：" + err.Error())
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = markdown.Excerpt(res.PlainText, 200)
	}
	slug, err := s.uniqueSlugFor(ctx, strings.TrimSpace(in.Slug), in.Title, existing.Slug)
	if err != nil {
		return nil, err
	}

	status := domain.StatusDraft
	if in.Publish {
		status = domain.StatusPublished
	}
	now := time.Now().UTC()
	// 首次发布才填 published_at；之后改草稿/重新发布都保留原值
	pubAt := existing.PublishedAt
	if in.Publish && !existing.PublishedAt.Valid {
		pubAt = nullTime(now)
	}

	if err := s.q.UpdatePost(ctx, repository.UpdatePostParams{
		Slug:           slug,
		Title:          in.Title,
		Summary:        summary,
		ContentMd:      in.ContentMd,
		ContentHtml:    res.HTML,
		TocJson:        tocNullString(res.TOC),
		CoverUrl:       strings.TrimSpace(in.CoverURL),
		Status:         int8(status),
		CommentEnabled: boolToInt8(in.CommentEnabled),
		ReadingTimeMin: readingTime(res.PlainText),
		PublishedAt:    pubAt,
		UpdatedAt:      now,
		ID:             existing.ID,
	}); err != nil {
		return nil, fmt.Errorf("更新文章: %w", err)
	}
	if err := s.linkTags(ctx, existing.ID, in.Tags); err != nil {
		return nil, err
	}
	return s.GetForEdit(ctx, actor, publicID)
}

// SetStatus 切换发布/下线。下线保留 published_at，便于将来重新发布。
func (s *AdminPostService) SetStatus(ctx context.Context, actor Actor, publicID string, publish bool) error {
	existing, err := s.q.GetPostForEdit(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPostNotFound
		}
		return fmt.Errorf("查询文章: %w", err)
	}
	if !actor.CanTouch(existing.AuthorID) {
		return ErrNotOwner
	}
	status := domain.StatusDraft
	if publish {
		status = domain.StatusPublished
	}
	pubAt := existing.PublishedAt
	now := time.Now().UTC()
	if publish && !existing.PublishedAt.Valid {
		pubAt = nullTime(now)
	}
	if err := s.q.UpdatePostStatus(ctx, repository.UpdatePostStatusParams{
		Status:      int8(status),
		PublishedAt: pubAt,
		UpdatedAt:   now,
		ID:          existing.ID,
	}); err != nil {
		return fmt.Errorf("切换状态: %w", err)
	}
	return nil
}

// SetCommentEnabled 开关评论。
func (s *AdminPostService) SetCommentEnabled(ctx context.Context, actor Actor, publicID string, enabled bool) error {
	existing, err := s.q.GetPostForEdit(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPostNotFound
		}
		return fmt.Errorf("查询文章: %w", err)
	}
	if !actor.CanTouch(existing.AuthorID) {
		return ErrNotOwner
	}
	if err := s.q.UpdatePostCommentEnabled(ctx, repository.UpdatePostCommentEnabledParams{
		CommentEnabled: boolToInt8(enabled),
		UpdatedAt:      time.Now().UTC(),
		ID:             existing.ID,
	}); err != nil {
		return fmt.Errorf("切换评论开关: %w", err)
	}
	return nil
}

// Delete 删除文章及其标签关联。
func (s *AdminPostService) Delete(ctx context.Context, actor Actor, publicID string) error {
	existing, err := s.q.GetPostForEdit(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPostNotFound
		}
		return fmt.Errorf("查询待删除文章: %w", err)
	}
	if !actor.CanTouch(existing.AuthorID) {
		return ErrNotOwner
	}
	// 先清标签关联，避免留下孤儿行（post_tags 无级联删除）
	if err := s.q.DeletePostTags(ctx, existing.ID); err != nil {
		return fmt.Errorf("删除标签关联: %w", err)
	}
	if err := s.q.DeletePostByID(ctx, existing.ID); err != nil {
		return fmt.Errorf("删除文章: %w", err)
	}
	return nil
}

// List 返回工作台的文章列表（含草稿）。
//
// 每个人（包括站点管理员）只能看到自己的文章（见 Actor.AuthorFilter）。
func (s *AdminPostService) List(ctx context.Context, actor Actor, statusFilter int8, page, pageSize int) (*domain.Page[domain.AdminPostListItem], error) {
	page, pageSize = normalizePage(page, pageSize)
	author := actor.AuthorFilter()

	rows, err := s.q.ListAdminPosts(ctx, repository.ListAdminPostsParams{
		Column1:  int8(statusFilter), // 0 表示不过滤
		Status:   statusFilter,
		Column3:  author,
		AuthorID: author,
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询后台文章列表: %w", err)
	}
	if statusFilter == 0 {
		// 统计总数也要忽略过滤条件
		total, err := s.q.CountAdminPosts(ctx, repository.CountAdminPostsParams{Column1: int8(0), Status: 0, Column3: author, AuthorID: author})
		if err != nil {
			return nil, fmt.Errorf("统计文章数: %w", err)
		}
		return s.toAdminPage(rows, total, page, pageSize), nil
	}
	total, err := s.q.CountAdminPosts(ctx, repository.CountAdminPostsParams{Column1: statusFilter, Status: statusFilter, Column3: author, AuthorID: author})
	if err != nil {
		return nil, fmt.Errorf("统计文章数: %w", err)
	}
	return s.toAdminPage(rows, total, page, pageSize), nil
}

// GetForEdit 返回编辑页需要的完整文章（含正文原文与标签）。
func (s *AdminPostService) GetForEdit(ctx context.Context, actor Actor, publicID string) (*domain.AdminPostDetail, error) {
	row, err := s.q.GetPostForEdit(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("查询文章: %w", err)
	}
	if !actor.CanTouch(row.AuthorID) {
		return nil, ErrNotOwner
	}
	tags, err := s.tagsOf(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &domain.AdminPostDetail{
		PublicID:       row.PublicID,
		Slug:           row.Slug,
		Title:          row.Title,
		Summary:        row.Summary,
		ContentMd:      row.ContentMd,
		CoverURL:       row.CoverUrl,
		Status:         domain.PostStatus(row.Status),
		CommentEnabled: row.CommentEnabled != 0,
		PublishedAt:    nullTimePtr(row.PublishedAt),
		UpdatedAt:      row.UpdatedAt,
		Tags:           tags,
	}, nil
}

// --- 内部 ---

func (s *AdminPostService) toAdminPage(rows []repository.ListAdminPostsRow, total int64, page, pageSize int) *domain.Page[domain.AdminPostListItem] {
	items := make([]domain.AdminPostListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.AdminPostListItem{
			PublicID:       r.PublicID,
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			CoverURL:       r.CoverUrl,
			Status:         domain.PostStatus(r.Status),
			CommentEnabled: r.CommentEnabled != 0,
			ReadingTimeMin: int(r.ReadingTimeMin),
			PublishedAt:    nullTimePtr(r.PublishedAt),
			UpdatedAt:      r.UpdatedAt,
		})
	}
	return &domain.Page[domain.AdminPostListItem]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// uniqueSlug 在新建时生成唯一 slug（与原 slug 永不冲突）。
func (s *AdminPostService) uniqueSlug(ctx context.Context, provided, title string) (string, error) {
	return s.uniqueSlugFor(ctx, provided, title, "")
}

// uniqueSlugFor 生成唯一 slug，但允许与 keepSlug 相同（即编辑自身）。
func (s *AdminPostService) uniqueSlugFor(ctx context.Context, provided, title, keepSlug string) (string, error) {
	base := strings.TrimSpace(provided)
	if base == "" {
		base = slug.Make(title)
	}
	base = strings.Trim(base, "-")
	if base == "" {
		base = "post"
	}
	out, ok := slug.EnsureUnique(base, func(c string) bool {
		if c == keepSlug {
			return false // 自己占用的 slug 不算冲突
		}
		n, err := s.q.CountPostBySlug(ctx, c)
		return err == nil && n > 0
	})
	if !ok {
		return "", apierr.InvalidArgument("无法生成唯一 slug，请手动指定")
	}
	return out, nil
}

// linkTags 先把文章的所有标签关联清空，再按输入重新关联（幂等）。
func (s *AdminPostService) linkTags(ctx context.Context, postID uint64, raw []string) error {
	seen := make(map[string]struct{})
	for _, t := range raw {
		name := strings.TrimSpace(t)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		id, err := s.q.UpsertTag(ctx, repository.UpsertTagParams{Slug: slug.Make(name), Name: name})
		if err != nil {
			return fmt.Errorf("写入标签 %q: %w", name, err)
		}
		if err := s.q.LinkPostTag(ctx, repository.LinkPostTagParams{PostID: postID, TagID: uint64(id)}); err != nil {
			return fmt.Errorf("关联标签 %q: %w", name, err)
		}
	}
	return nil
}

func (s *AdminPostService) tagsOf(ctx context.Context, postID uint64) ([]string, error) {
	rows, err := s.q.ListTagsByPostID(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("查询文章标签: %w", err)
	}
	out := make([]string, 0, len(rows))
	for _, t := range rows {
		out = append(out, t.Name)
	}
	return out, nil
}
