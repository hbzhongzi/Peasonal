// Package service 承载业务规则。
//
// 单向依赖：transport → service → repository。
// service 不感知 HTTP（不 import net/http），repository 不感知业务规则。
// 这样将来加 CLI 工具（批量导入、重建索引）不需要拖着 HTTP 层。
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"personal-blog/internal/domain"
	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/pkg/markdown"
	"personal-blog/internal/repository"
)

// 分页边界。限制上限是因为 OFFSET 分页在大偏移量下会退化，
// 而这个博客的文章量根本用不到更大的页。
const (
	DefaultPageSize = 10
	MaxPageSize     = 50
)

// ErrPostNotFound 由传输层转成 404。
//
// 直接用 *apierr.Error 而不是裸 error：业务层最清楚「什么叫没找到」，
// 让传输层再判一次等于把同一条规则写两遍。apierr 只有 HTTP 状态码语义，
// 不含 HTTP 实现细节，service 依赖它不算越界。
var ErrPostNotFound = apierr.NotFound("文章不存在或未发布")

// PostService 提供文章的读路径。
type PostService struct {
	q *repository.Queries
}

func NewPostService(q *repository.Queries) *PostService {
	return &PostService{q: q}
}

// List 返回已发布文章的分页列表。
func (s *PostService) List(ctx context.Context, page, pageSize int) (*domain.Page[domain.PostListItem], error) {
	page, pageSize = normalizePage(page, pageSize)

	total, err := s.q.CountPublishedPosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计文章数: %w", err)
	}

	rows, err := s.q.ListPublishedPosts(ctx, repository.ListPublishedPostsParams{
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询文章列表: %w", err)
	}

	items, ids, authorIDs := toListItems(rows)
	if err := s.attachTags(ctx, items, ids); err != nil {
		return nil, err
	}
	if err := s.attachAuthors(ctx, items, authorIDs); err != nil {
		return nil, err
	}

	return newPage(items, total, page, pageSize), nil
}

// GetBySlug 返回已发布文章详情。
func (s *PostService) GetBySlug(ctx context.Context, slug string) (*domain.Post, error) {
	row, err := s.q.GetPostBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("查询文章: %w", err)
	}
	return s.toPost(ctx, row)
}

// Search 在已发布文章里做 ngram 全文检索（标题 + 正文）。
//
// 空查询直接返回空结果，避免 MySQL 把空串当「匹配全部」处理。
func (s *PostService) Search(ctx context.Context, q string, page, pageSize int) (*domain.Page[domain.PostListItem], error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return newPage([]domain.PostListItem{}, 0, page, pageSize), nil
	}
	page, pageSize = normalizePage(page, pageSize)

	total, err := s.q.CountSearchPublishedPosts(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("统计搜索结果: %w", err)
	}
	if total == 0 {
		return newPage([]domain.PostListItem{}, 0, page, pageSize), nil
	}

	rows, err := s.q.SearchPublishedPosts(ctx, repository.SearchPublishedPostsParams{
		Column1: q,
		Column2: q,
		Limit:   int32(pageSize),
		Offset:  int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("搜索文章: %w", err)
	}

	items, ids, authorIDs := toSearchListItems(rows)
	if err := s.attachTags(ctx, items, ids); err != nil {
		return nil, err
	}
	if err := s.attachAuthors(ctx, items, authorIDs); err != nil {
		return nil, err
	}
	return newPage(items, total, page, pageSize), nil
}

// ListByTag 返回某个标签下的已发布文章。
func (s *PostService) ListByTag(ctx context.Context, tagSlug string, page, pageSize int) (*domain.Page[domain.PostListItem], error) {
	page, pageSize = normalizePage(page, pageSize)

	total, err := s.q.CountPublishedPostsByTag(ctx, tagSlug)
	if err != nil {
		return nil, fmt.Errorf("统计标签文章数: %w", err)
	}

	rows, err := s.q.ListPublishedPostsByTag(ctx, repository.ListPublishedPostsByTagParams{
		Slug:   tagSlug,
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询标签文章: %w", err)
	}

	items, ids, authorIDs := toTagListItems(rows)
	if err := s.attachTags(ctx, items, ids); err != nil {
		return nil, err
	}
	if err := s.attachAuthors(ctx, items, authorIDs); err != nil {
		return nil, err
	}

	return newPage(items, total, page, pageSize), nil
}

// ListByAuthor 返回某个用户公开博客页的文章列表（只含已发布）。
func (s *PostService) ListByAuthor(ctx context.Context, authorPublicID string, page, pageSize int) (*domain.Page[domain.PostListItem], error) {
	page, pageSize = normalizePage(page, pageSize)

	total, err := s.q.CountPublishedPostsByAuthor(ctx, authorPublicID)
	if err != nil {
		return nil, fmt.Errorf("统计作者文章数: %w", err)
	}

	rows, err := s.q.ListPublishedPostsByAuthor(ctx, repository.ListPublishedPostsByAuthorParams{
		PublicID: authorPublicID,
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询作者文章: %w", err)
	}

	items, ids, authorIDs := toAuthorListItems(rows)
	if err := s.attachTags(ctx, items, ids); err != nil {
		return nil, err
	}
	if err := s.attachAuthors(ctx, items, authorIDs); err != nil {
		return nil, err
	}

	return newPage(items, total, page, pageSize), nil
}

// AuthorProfile 返回用户的公开资料（/users/{public_id}，匿名可读）。
func (s *PostService) AuthorProfile(ctx context.Context, authorPublicID string) (*domain.PublicProfile, error) {
	row, err := s.q.GetPublicUserByPublicID(ctx, authorPublicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("用户不存在")
		}
		return nil, fmt.Errorf("查询用户资料: %w", err)
	}
	return &domain.PublicProfile{
		PublicID:    row.PublicID,
		DisplayName: row.DisplayName,
		Bio:         row.Bio,
		AvatarURL:   row.AvatarUrl,
		Github:      row.Github,
	}, nil
}

// ListTags 返回全部有已发布文章的标签。
func (s *PostService) ListTags(ctx context.Context) ([]domain.TagWithCount, error) {
	rows, err := s.q.ListTagsWithCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询标签: %w", err)
	}
	out := make([]domain.TagWithCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.TagWithCount{
			Tag:       domain.Tag{Slug: r.Slug, Name: r.Name},
			PostCount: r.PostCount,
		})
	}
	return out, nil
}

// Archive 返回按月分组的归档索引。
func (s *PostService) Archive(ctx context.Context) ([]domain.ArchiveMonth, error) {
	rows, err := s.q.ListArchiveMonths(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询归档: %w", err)
	}
	out := make([]domain.ArchiveMonth, 0, len(rows))
	for _, r := range rows {
		m, ok := parseYearMonth(r.Ym)
		if !ok {
			// DATE_FORMAT 结果异常就跳过，别让一条脏数据把整页打挂
			slog.Warn("归档月份格式异常，已跳过", slog.String("ym", r.Ym))
			continue
		}
		out = append(out, domain.ArchiveMonth{
			Year:  m.year,
			Month: m.month,
			Label: fmt.Sprintf("%d 年 %d 月", m.year, m.month),
			Total: r.Total,
		})
	}
	return out, nil
}

// ListByMonth 返回某个月的已发布文章。
func (s *PostService) ListByMonth(ctx context.Context, year, month, page, pageSize int) (*domain.Page[domain.PostListItem], error) {
	if month < 1 || month > 12 {
		return nil, apierr.InvalidArgument("月份必须是 1-12")
	}
	if year < 1970 || year > 9999 {
		return nil, apierr.InvalidArgument("年份不合法")
	}
	page, pageSize = normalizePage(page, pageSize)

	// 半开区间 [start, end)，才能命中 (status, published_at) 索引
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	total, err := s.q.CountPublishedPostsByMonth(ctx, repository.CountPublishedPostsByMonthParams{
		PublishedAt:   nullTime(start),
		PublishedAt_2: nullTime(end),
	})
	if err != nil {
		return nil, fmt.Errorf("统计月份文章数: %w", err)
	}

	rows, err := s.q.ListPublishedPostsByMonth(ctx, repository.ListPublishedPostsByMonthParams{
		PublishedAt:   nullTime(start),
		PublishedAt_2: nullTime(end),
		Limit:         int32(pageSize),
		Offset:        int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询月份文章: %w", err)
	}

	items, ids, authorIDs := toMonthListItems(rows)
	if err := s.attachTags(ctx, items, ids); err != nil {
		return nil, err
	}
	if err := s.attachAuthors(ctx, items, authorIDs); err != nil {
		return nil, err
	}

	return newPage(items, total, page, pageSize), nil
}

// --- 内部 ---

// attachTags 批量补齐列表项的标签。
//
// 一次查询取回整页标签，避免 N+1 —— 列表页每页 10-20 篇，
// 逐篇查标签会让一次列表请求变成十几次数据库往返。
func (s *PostService) attachTags(ctx context.Context, items []domain.PostListItem, ids []uint64) error {
	if len(items) == 0 {
		return nil
	}
	tagRows, err := s.q.ListTagsByPostIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("批量查询标签: %w", err)
	}
	byPost := make(map[uint64][]domain.Tag, len(items))
	for _, t := range tagRows {
		byPost[t.PostID] = append(byPost[t.PostID], domain.Tag{Slug: t.Slug, Name: t.Name})
	}
	for i := range items {
		items[i].Tags = byPost[ids[i]]
	}
	return nil
}

// attachAuthors 批量补齐列表项的作者信息（与 attachTags 同思路，避免 N+1）。
//
// 某个作者被删了（理论上不会，但防御性考虑）对应项保持零值 —— 不能因为一个
// 作者缺席让整页文章打不开。
func (s *PostService) attachAuthors(ctx context.Context, items []domain.PostListItem, authorIDs []uint64) error {
	if len(items) == 0 {
		return nil
	}
	rows, err := s.q.ListPublicUsersByIDs(ctx, authorIDs)
	if err != nil {
		return fmt.Errorf("批量查询作者: %w", err)
	}
	byID := make(map[uint64]domain.PostAuthor, len(rows))
	for _, r := range rows {
		byID[r.ID] = domain.PostAuthor{PublicID: r.PublicID, DisplayName: r.DisplayName, AvatarURL: r.AvatarUrl}
	}
	for i := range items {
		items[i].Author = byID[authorIDs[i]]
	}
	return nil
}

func (s *PostService) toPost(ctx context.Context, row repository.GetPostBySlugRow) (*domain.Post, error) {
	tags, err := s.q.ListTagsByPostID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("查询文章标签: %w", err)
	}
	list := make([]domain.Tag, 0, len(tags))
	for _, t := range tags {
		list = append(list, domain.Tag{Slug: t.Slug, Name: t.Name})
	}

	// 作者资料查不到只影响署名，不影响文章本身：ErrNoRows 静默留空，其余照常报错
	author := domain.PostAuthor{}
	a, err := s.q.GetPublicUserByID(ctx, row.AuthorID)
	switch {
	case err == nil:
		author = domain.PostAuthor{PublicID: a.PublicID, DisplayName: a.DisplayName, AvatarURL: a.AvatarUrl}
	case errors.Is(err, sql.ErrNoRows):
		// 留空
	default:
		return nil, fmt.Errorf("查询文章作者: %w", err)
	}

	return &domain.Post{
		PublicID:       row.PublicID,
		Slug:           row.Slug,
		Title:          row.Title,
		Summary:        row.Summary,
		ContentHTML:    row.ContentHtml,
		TOC:            parseTOC(row.TocJson),
		CoverURL:       row.CoverUrl,
		ReadingTimeMin: int(row.ReadingTimeMin),
		CommentEnabled: row.CommentEnabled != 0,
		PublishedAt:    nullTimePtr(row.PublishedAt),
		Tags:           list,
		Author:         author,
	}, nil
}

// parseTOC 还原渲染时生成的目录。
//
// 目录损坏只影响导航、不影响正文，所以解析失败时返回空目录而不是让整个请求失败。
func parseTOC(raw sql.NullString) []markdown.TOCItem {
	if !raw.Valid || raw.String == "" {
		return []markdown.TOCItem{}
	}
	var toc []markdown.TOCItem
	if err := json.Unmarshal([]byte(raw.String), &toc); err != nil {
		slog.Warn("目录 JSON 解析失败，回退为空目录", slog.String("err", err.Error()))
		return []markdown.TOCItem{}
	}
	return toc
}

// 四个 toXxxListItems 长得像，但 sqlc 为每个查询生成了独立的 Row 类型，
// 与其做泛型体操，不如老老实实写四遍 —— 各 8 行，改起来也更直观。
// 第三个返回值是 author_id 列表，供 attachAuthors 批量补作者。
func toListItems(rows []repository.ListPublishedPostsRow) ([]domain.PostListItem, []uint64, []uint64) {
	items := make([]domain.PostListItem, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	authorIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.PostListItem{
			PublicID:       r.PublicID,
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			CoverURL:       r.CoverUrl,
			ReadingTimeMin: int(r.ReadingTimeMin),
			PublishedAt:    nullTimePtr(r.PublishedAt),
		})
		ids = append(ids, r.ID)
		authorIDs = append(authorIDs, r.AuthorID)
	}
	return items, ids, authorIDs
}

func toTagListItems(rows []repository.ListPublishedPostsByTagRow) ([]domain.PostListItem, []uint64, []uint64) {
	items := make([]domain.PostListItem, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	authorIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.PostListItem{
			PublicID:       r.PublicID,
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			CoverURL:       r.CoverUrl,
			ReadingTimeMin: int(r.ReadingTimeMin),
			PublishedAt:    nullTimePtr(r.PublishedAt),
		})
		ids = append(ids, r.ID)
		authorIDs = append(authorIDs, r.AuthorID)
	}
	return items, ids, authorIDs
}

func toMonthListItems(rows []repository.ListPublishedPostsByMonthRow) ([]domain.PostListItem, []uint64, []uint64) {
	items := make([]domain.PostListItem, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	authorIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.PostListItem{
			PublicID:       r.PublicID,
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			CoverURL:       r.CoverUrl,
			ReadingTimeMin: int(r.ReadingTimeMin),
			PublishedAt:    nullTimePtr(r.PublishedAt),
		})
		ids = append(ids, r.ID)
		authorIDs = append(authorIDs, r.AuthorID)
	}
	return items, ids, authorIDs
}

func toSearchListItems(rows []repository.SearchPublishedPostsRow) ([]domain.PostListItem, []uint64, []uint64) {
	items := make([]domain.PostListItem, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	authorIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.PostListItem{
			PublicID:       r.PublicID,
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			CoverURL:       r.CoverUrl,
			ReadingTimeMin: int(r.ReadingTimeMin),
			PublishedAt:    nullTimePtr(r.PublishedAt),
		})
		ids = append(ids, r.ID)
		authorIDs = append(authorIDs, r.AuthorID)
	}
	return items, ids, authorIDs
}

func toAuthorListItems(rows []repository.ListPublishedPostsByAuthorRow) ([]domain.PostListItem, []uint64, []uint64) {
	items := make([]domain.PostListItem, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	authorIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		items = append(items, domain.PostListItem{
			PublicID:       r.PublicID,
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			CoverURL:       r.CoverUrl,
			ReadingTimeMin: int(r.ReadingTimeMin),
			PublishedAt:    nullTimePtr(r.PublishedAt),
		})
		ids = append(ids, r.ID)
		authorIDs = append(authorIDs, r.AuthorID)
	}
	return items, ids, authorIDs
}

func newPage(items []domain.PostListItem, total int64, page, pageSize int) *domain.Page[domain.PostListItem] {
	return &domain.Page[domain.PostListItem]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

// normalizePage 收敛分页参数：越界值一律拉回合法区间，而不是报错。
// 访客手改 URL 里的 page=0 或 page_size=9999 不该得到 400。
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

type yearMonth struct {
	year  int
	month int
}

func parseYearMonth(ym string) (yearMonth, bool) {
	// 形如 "2026-09"
	if len(ym) != 7 || ym[4] != '-' {
		return yearMonth{}, false
	}
	y, err := strconv.Atoi(ym[:4])
	if err != nil {
		return yearMonth{}, false
	}
	m, err := strconv.Atoi(ym[5:])
	if err != nil || m < 1 || m > 12 {
		return yearMonth{}, false
	}
	return yearMonth{year: y, month: m}, true
}
