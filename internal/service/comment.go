package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"personal-blog/internal/domain"
	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/repository"
)

// 评论防垃圾四道闸（对应设计文档 §7.5 / M3-02）：
//  1. 蜜罐字段：机器人填了就静默丢弃，返回成功假象
//  2. 最小间隔：同一 IP 10 秒内不能连发
//  3. 频率：同一 IP 每分钟最多 1 条
//  4. 日上限：同一 IP 每天最多 50 条（简化实现）
//
// ip 只存加盐哈希，满足隐私合规，也避免明文入库。
const (
	commentMinInterval = 10 * time.Second
	commentWindow      = time.Minute
	commentDailyCap    = 50
	commentMaxLen      = 2000
	authorMaxLen       = 64
)

// CommentService 提供评论的写入与读取。
type CommentService struct {
	q      *repository.Queries
	pepper string // 用于给 IP 加盐哈希，取自服务端密钥
}

func NewCommentService(q *repository.Queries, pepper string) *CommentService {
	return &CommentService{q: q, pepper: pepper}
}

// Create 提交一条评论。无审核，提交即通过（status=1）。
func (s *CommentService) Create(ctx context.Context, postSlug, clientIP string, in domain.CommentInput) (*domain.Comment, error) {
	// 1) 蜜罐
	if strings.TrimSpace(in.Honeypot) != "" {
		// 静默丢弃，返回伪装成功（别告诉机器人被识破）
		return &domain.Comment{PublicID: newULID()}, nil
	}

	author := strings.TrimSpace(in.AuthorName)
	content := strings.TrimSpace(in.Content)
	if author == "" {
		return nil, apierr.InvalidArgument("请填写昵称")
	}
	if len([]rune(author)) > authorMaxLen {
		return nil, apierr.InvalidArgument("昵称过长")
	}
	if content == "" {
		return nil, apierr.InvalidArgument("评论内容不能为空")
	}
	if len([]rune(content)) > commentMaxLen {
		return nil, apierr.InvalidArgument("评论内容过长")
	}

	// 文章必须存在且已发布、且开放评论
	post, err := s.q.GetPostIDBySlug(ctx, postSlug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("文章不存在或尚未发布")
		}
		return nil, fmt.Errorf("查询文章: %w", err)
	}
	if post.CommentEnabled == 0 {
		return nil, apierr.Forbidden("该文章已关闭评论")
	}

	ipHash := hashIP(clientIP, s.pepper)
	now := time.Now().UTC()

	// 2) 最小间隔
	if latest, err := s.q.LatestCommentByIP(ctx, ipHash); err == nil {
		if now.Sub(latest) < commentMinInterval {
			return nil, apierr.RateLimited("评论太快了，请稍后再试")
		}
	}
	// 3) 每分钟限 1 条
	if n, err := s.q.CountRecentCommentsByIP(ctx, repository.CountRecentCommentsByIPParams{
		IpHash:    ipHash,
		CreatedAt: now.Add(-commentWindow),
	}); err == nil && n >= 1 {
		return nil, apierr.RateLimited("评论过于频繁，请稍后再试")
	}
	// 4) 每日上限
	if n, err := s.q.CountRecentCommentsByIP(ctx, repository.CountRecentCommentsByIPParams{
		IpHash:    ipHash,
		CreatedAt: now.Add(-24 * time.Hour),
	}); err == nil && n >= commentDailyCap {
		return nil, apierr.RateLimited("今日评论已达上限")
	}

	pid := newULID()
	if _, err := s.q.CreateComment(ctx, repository.CreateCommentParams{
		PublicID:    pid,
		PostID:      post.ID,
		AuthorName:  author,
		AuthorEmail: strings.TrimSpace(in.AuthorEmail),
		Content:     content,
		Status:      1,
		IpHash:      ipHash,
	}); err != nil {
		return nil, fmt.Errorf("写入评论: %w", err)
	}
	return &domain.Comment{
		PublicID:   pid,
		AuthorName: author,
		Content:    content,
		CreatedAt:  now,
	}, nil
}

// PostIDBySlug 把 slug 解析成 post id（后台按文章查评论用）。
func (s *CommentService) PostIDBySlug(ctx context.Context, slug string) (uint64, error) {
	post, err := s.q.GetPostIDBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apierr.NotFound("文章不存在或尚未发布")
		}
		return 0, fmt.Errorf("查询文章: %w", err)
	}
	return post.ID, nil
}

// ListPublicPaginated 返回某篇文章的公开评论（按时间正序，支持分页）。
// 首页用旧评在前、新评在后，契合阅读流；滑动加载更多时翻下一页即可。
func (s *CommentService) ListPublicPaginated(ctx context.Context, postSlug string, page, pageSize int) (*domain.Page[domain.Comment], error) {
	page, pageSize = normalizePage(page, pageSize)
	post, err := s.q.GetPostIDBySlug(ctx, postSlug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("文章不存在")
		}
		return nil, fmt.Errorf("查询文章: %w", err)
	}
	rows, err := s.q.ListPublicCommentsByPostPaginated(ctx, repository.ListPublicCommentsByPostPaginatedParams{
		PostID: post.ID,
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询评论: %w", err)
	}
	total, err := s.q.CountCommentsByPost(ctx, post.ID)
	if err != nil {
		return nil, fmt.Errorf("统计评论: %w", err)
	}
	out := make([]domain.Comment, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Comment{
			PublicID:   r.PublicID,
			AuthorName: r.AuthorName,
			Content:    r.Content,
			CreatedAt:  r.CreatedAt,
		})
	}
	return &domain.Page[domain.Comment]{
		Items:    out,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CountPublic 返回某篇文章的公开评论总数（用于评论图标上的计数，不拉取内容）。
func (s *CommentService) CountPublic(ctx context.Context, postSlug string) (int64, error) {
	post, err := s.q.GetPostIDBySlug(ctx, postSlug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apierr.NotFound("文章不存在")
		}
		return 0, fmt.Errorf("查询文章: %w", err)
	}
	total, err := s.q.CountCommentsByPost(ctx, post.ID)
	if err != nil {
		return 0, fmt.Errorf("统计评论: %w", err)
	}
	return total, nil
}

// ListByPostAdmin 返回后台管理用的评论列表（含已隐藏）。
func (s *CommentService) ListByPostAdmin(ctx context.Context, postID uint64, page, pageSize int) (*domain.Page[domain.AdminComment], error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, err := s.q.ListCommentsByPostAny(ctx, repository.ListCommentsByPostAnyParams{
		PostID: postID,
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询评论: %w", err)
	}
	total, err := s.q.CountCommentsByPostAny(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("统计评论: %w", err)
	}
	out := make([]domain.AdminComment, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AdminComment{
			PublicID:   r.PublicID,
			AuthorName: r.AuthorName,
			Content:    r.Content,
			Status:     r.Status,
			CreatedAt:  r.CreatedAt,
		})
	}
	return &domain.Page[domain.AdminComment]{
		Items:    out,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Hide 隐藏一条评论（保留记录）。
func (s *CommentService) Hide(ctx context.Context, publicID string) error {
	if _, err := s.q.GetCommentByPublicID(ctx, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apierr.NotFound("评论不存在")
		}
		return fmt.Errorf("查询评论: %w", err)
	}
	if err := s.q.HideCommentByPublicID(ctx, publicID); err != nil {
		return fmt.Errorf("隐藏评论: %w", err)
	}
	return nil
}

// Delete 删除一条评论。
func (s *CommentService) Delete(ctx context.Context, publicID string) error {
	if _, err := s.q.GetCommentByPublicID(ctx, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apierr.NotFound("评论不存在")
		}
		return fmt.Errorf("查询评论: %w", err)
	}
	if err := s.q.DeleteCommentByPublicID(ctx, publicID); err != nil {
		return fmt.Errorf("删除评论: %w", err)
	}
	return nil
}

// hashIP 对客户端 IP 加盐哈希，避免明文入库。
func hashIP(ip, pepper string) string {
	sum := sha256.Sum256([]byte(ip + ":" + pepper))
	return hex.EncodeToString(sum[:])
}
