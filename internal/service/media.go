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
	"personal-blog/internal/pkg/media"
	"personal-blog/internal/repository"
)

// MediaService 负责图片上传、列表与删除。
//
// 文件落盘与处理在 media.Store（见 §5.5）；这里只负责配额校验与元数据读写。
type MediaService struct {
	q         *repository.Queries
	store     *media.Store
	quota     int64 // 配额（字节）
	urlPrefix string
}

func NewMediaService(q *repository.Queries, store *media.Store, quotaMB int64, urlPrefix string) *MediaService {
	return &MediaService{
		q:         q,
		store:     store,
		quota:     quotaMB * 1024 * 1024,
		urlPrefix: strings.TrimSuffix(urlPrefix, "/"),
	}
}

// Upload 处理并保存一张图片，返回其元数据。data 已由 handler 用 MaxBytesReader 限流。
//
// 归属写进 author_id：媒体库按人隔离，谁传的谁能看/能删（管理员除外）。
// 不写归属就会出现「所有人共用一张图列表、还能互删」的隐私问题。
func (s *MediaService) Upload(ctx context.Context, actor Actor, data []byte) (*domain.MediaItem, error) {
	used, err := s.q.SumMediaSize(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计媒体用量: %w", err)
	}
	if int64(len(data)) > s.quota-used {
		return nil, apierr.InvalidArgument("媒体库空间不足，请先清理旧文件")
	}

	res, err := s.store.Save(data)
	if err != nil {
		// ErrUnsupportedType 直接转成可理解的 400，其余按 400（处理失败）
		if errors.Is(err, media.ErrUnsupportedType) {
			return nil, apierr.InvalidArgument("不支持的图片格式（仅接受 JPEG/PNG/GIF/WebP）")
		}
		return nil, apierr.InvalidArgument("图片处理失败：" + err.Error())
	}

	pid := newULID()
	if _, err := s.q.CreateMedia(ctx, repository.CreateMediaParams{
		PublicID:  pid,
		Url:       res.URL,
		Filename:  res.Filename,
		SizeBytes: uint64(res.SizeBytes),
		MimeType:  res.MimeType,
		Width:     uint32(res.Width),
		Height:    uint32(res.Height),
		AuthorID:  actor.UserID,
	}); err != nil {
		return nil, fmt.Errorf("写入媒体元数据: %w", err)
	}
	return &domain.MediaItem{
		PublicID:  pid,
		URL:       res.URL,
		Filename:  res.Filename,
		SizeBytes: uint64(res.SizeBytes),
		MimeType:  res.MimeType,
		Width:     uint32(res.Width),
		Height:    uint32(res.Height),
		CreatedAt: time.Now().UTC(),
	}, nil
}

// List 返回媒体库分页列表（最新在前）。
//
// 只列出本人上传的图片（包括站点管理员，在「我的博客」里也不能看他人的媒体库）。
func (s *MediaService) List(ctx context.Context, actor Actor, page, pageSize int) (*domain.Page[domain.MediaItem], error) {
	page, pageSize = normalizePage(page, pageSize)
	author := actor.AuthorFilter()
	rows, err := s.q.ListMedia(ctx, repository.ListMediaParams{
		Column1:  author,
		AuthorID: author,
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("查询媒体库: %w", err)
	}
	total, err := s.q.CountMedia(ctx, repository.CountMediaParams{Column1: author, AuthorID: author})
	if err != nil {
		return nil, fmt.Errorf("统计媒体: %w", err)
	}
	out := make([]domain.MediaItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.MediaItem{
			PublicID:  r.PublicID,
			URL:       r.Url,
			Filename:  r.Filename,
			SizeBytes: r.SizeBytes,
			MimeType:  r.MimeType,
			Width:     r.Width,
			Height:    r.Height,
			CreatedAt: r.CreatedAt,
		})
	}
	return &domain.Page[domain.MediaItem]{
		Items:    out,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Delete 删除文件与元数据。先删磁盘文件，再删库记录。
func (s *MediaService) Delete(ctx context.Context, actor Actor, publicID string) error {
	m, err := s.q.GetMediaByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apierr.NotFound("媒体不存在")
		}
		return fmt.Errorf("查询媒体: %w", err)
	}
	if !actor.CanTouch(m.AuthorID) {
		return ErrNotOwner
	}
	rel := strings.TrimPrefix(m.Url, s.urlPrefix)
	rel = strings.TrimPrefix(rel, "/")
	if err := s.store.Delete(rel); err != nil {
		return fmt.Errorf("删除文件: %w", err)
	}
	if err := s.q.DeleteMediaByPublicID(ctx, publicID); err != nil {
		return fmt.Errorf("删除元数据: %w", err)
	}
	return nil
}
