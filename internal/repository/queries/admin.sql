-- 后台文章查询。与公开接口的关键区别：
-- 1. 不过滤 status —— 后台必须能看见草稿
-- 2. 取 content_md（编辑要用原文），但不取 content_html（几十 KB，编辑页用不上）
--
-- 注意：新建文章用的是 writing.sql 的 CreatePost（写路径共用，含 created_at/updated_at）。

-- name: ListAdminPosts :many
-- status=0 表示不过滤；author_filter=0 表示不过滤作者（管理员），
-- 否则只看该作者的文章（普通作者只能看见自己的，含草稿）。
-- 用这种写法而不是动态拼 SQL，是为了让 sqlc 能静态分析。
SELECT id, public_id, slug, title, summary, cover_url, status, comment_enabled,
       reading_time_min, published_at, created_at, updated_at
FROM posts
WHERE (? = 0 OR status = ?)
  AND (? = 0 OR author_id = ?)
ORDER BY COALESCE(published_at, created_at) DESC
LIMIT ? OFFSET ?;

-- name: CountAdminPosts :one
SELECT COUNT(*) AS total
FROM posts
WHERE (? = 0 OR status = ?)
  AND (? = 0 OR author_id = ?);

-- name: GetPostForEdit :one
-- author_id 用于归属校验：只有作者本人（或管理员）能改这篇文章。
SELECT id, public_id, slug, title, summary, content_md, cover_url, status,
       comment_enabled, published_at, updated_at, author_id
FROM posts
WHERE public_id = ?
LIMIT 1;

-- name: CountPostBySlug :one
-- 新建/改名时校验 slug 唯一性（slug 列上有唯一索引）
SELECT COUNT(*) AS total
FROM posts
WHERE slug = ?;

-- name: UpdatePost :exec
UPDATE posts
SET slug = ?, title = ?, summary = ?, content_md = ?, content_html = ?, toc_json = ?,
    cover_url = ?, status = ?, comment_enabled = ?, reading_time_min = ?, published_at = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdatePostStatus :exec
-- published_at 由 service 决定：首次发布填当前时间，下线时保留原值
UPDATE posts
SET status = ?, published_at = ?, updated_at = ?
WHERE id = ?;

-- name: UpdatePostCommentEnabled :exec
UPDATE posts
SET comment_enabled = ?, updated_at = ?
WHERE id = ?;

-- name: DeletePostByID :exec
DELETE FROM posts
WHERE id = ?;
