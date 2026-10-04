-- 评论查询。本期无审核：提交即 status=1 公开（设计文档 §7.5）。

-- name: ListPublicCommentsByPost :many
-- 只取公开评论；author_email 绝不可出现在结果里（隐私）
SELECT id, public_id, author_name, content, created_at
FROM comments
WHERE post_id = ? AND status = 1
ORDER BY created_at ASC, id ASC;

-- name: ListPublicCommentsByPostPaginated :many
-- 公开评论分页版（首页按时间正序，旧评在前、新评在后，便于阅读流）。
-- LIMIT/OFFSET 由 service 计算；author_email 同样不返回。
SELECT id, public_id, author_name, content, created_at
FROM comments
WHERE post_id = ? AND status = 1
ORDER BY created_at ASC, id ASC
LIMIT ? OFFSET ?;

-- name: ListCommentsByPostAny :many
-- 后台用：含已隐藏的
SELECT id, public_id, author_name, content, status, created_at
FROM comments
WHERE post_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountCommentsByPost :one
SELECT COUNT(*) AS total
FROM comments
WHERE post_id = ? AND status = 1;

-- name: CountCommentsByPostAny :one
-- 后台用：含已隐藏的，便于分页总数正确
SELECT COUNT(*) AS total
FROM comments
WHERE post_id = ?;

-- name: CreateComment :execlastid
INSERT INTO comments (public_id, post_id, author_name, author_email, content, status, ip_hash)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetCommentByPublicID :one
SELECT id, public_id, post_id, author_name, content, status, created_at
FROM comments
WHERE public_id = ?
LIMIT 1;

-- name: HideCommentByPublicID :exec
-- 隐藏而不是删除：保留记录便于回溯是谁、什么时候发的
UPDATE comments
SET status = 2
WHERE public_id = ?;

-- name: DeleteCommentByPublicID :exec
DELETE FROM comments
WHERE public_id = ?;

-- name: GetPostIDBySlug :one
-- 提交评论前要确认文章存在且已发布（不能给草稿发评论）
SELECT id, comment_enabled
FROM posts
WHERE slug = ? AND status = 2
LIMIT 1;

-- name: CountRecentCommentsByIP :one
-- 限流：同一 IP 在窗口期内的评论数
SELECT COUNT(*) AS total
FROM comments
WHERE ip_hash = ? AND created_at >= ?;

-- name: LatestCommentByIP :one
-- 最小间隔校验取最近一条的时间
SELECT created_at
FROM comments
WHERE ip_hash = ?
ORDER BY created_at DESC
LIMIT 1;
