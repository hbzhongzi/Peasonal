-- 写路径查询。M1 由 cmd/seed 使用，M2 的后台复用这里的大部分语句。
--
-- 注意：这些语句都不带 status 过滤 —— 写操作必须能看到草稿。

-- name: GetPostBySlugAny :one
SELECT id, public_id, slug, title, status
FROM posts
WHERE slug = ?
LIMIT 1;

-- name: CreatePost :execlastid
INSERT INTO posts (public_id, slug, title, summary, content_md, content_html, toc_json,
                   cover_url, status, comment_enabled, reading_time_min,
                   published_at, created_at, updated_at, author_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePostByID :exec
UPDATE posts
SET title = ?, summary = ?, content_md = ?, content_html = ?, toc_json = ?,
    cover_url = ?, reading_time_min = ?, published_at = ?, updated_at = ?
WHERE id = ?;

-- name: CreateTag :execlastid
-- 标签计数不落库，由 ListTagsWithCount 实时 JOIN 统计（见迁移 00019）
INSERT INTO tags (slug, name)
VALUES (?, ?);

-- name: UpsertTag :execlastid
-- 后台编辑时标签可能已存在：存在则复用其 id，不存在则插入。
-- ON DUPLICATE KEY UPDATE 配 LAST_INSERT_ID(id) 让「已存在」也能拿到该行 id。
INSERT INTO tags (slug, name) VALUES (?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id), name = VALUES(name);

-- name: DeletePostTags :exec
DELETE FROM post_tags
WHERE post_id = ?;

-- name: LinkPostTag :exec
-- IGNORE 是为了容忍重复关联（重跑 seed 时很常见）
INSERT IGNORE INTO post_tags (post_id, tag_id)
VALUES (?, ?);

-- name: CreateUser :execlastid
INSERT INTO users (public_id, email, password_hash, display_name, can_upload, can_publish, is_admin)
VALUES (?, ?, ?, ?, ?, ?, ?);
