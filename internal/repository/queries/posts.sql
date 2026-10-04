-- name: ListPublishedPosts :many
-- 列表页只取摘要字段，不取 content_md / content_html（正文能到几十 KB）。
-- author_id 用于批量补作者信息（见 users.ListPublicUsersByIDs），不在本查询 JOIN。
SELECT id, public_id, slug, title, summary, cover_url, reading_time_min, published_at, author_id
FROM posts
WHERE status = 2
ORDER BY published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedPosts :one
SELECT COUNT(*) AS total
FROM posts
WHERE status = 2;

-- name: GetPostBySlug :one
SELECT id, public_id, slug, title, summary, content_md, content_html, toc_json, cover_url,
       status, comment_enabled, reading_time_min, published_at, created_at, updated_at, author_id
FROM posts
WHERE slug = ? AND status = 2
LIMIT 1;

-- name: GetPostByPublicID :one
SELECT id, public_id, slug, title, summary, content_md, content_html, toc_json, cover_url,
       status, comment_enabled, reading_time_min, published_at, created_at, updated_at, author_id
FROM posts
WHERE public_id = ?
LIMIT 1;

-- name: ListTagsByPostID :many
SELECT t.id, t.slug, t.name
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
WHERE pt.post_id = ?
ORDER BY t.name;

-- name: ListTagsByPostIDs :many
-- 列表页用一次批量查询取回全部标签，避免 N+1。
-- 不用 GROUP_CONCAT：它有 group_concat_max_len（默认 1024 字节）截断风险。
SELECT pt.post_id, t.slug, t.name
FROM post_tags pt
JOIN tags t ON t.id = pt.tag_id
WHERE pt.post_id IN (sqlc.slice('post_ids'))
ORDER BY t.name;

-- name: ListTagsWithCount :many
-- 只统计已发布文章：草稿挂着的标签不该让标签凭空出现在前台
SELECT t.id, t.slug, t.name, COUNT(pt.post_id) AS post_count
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
JOIN posts p ON p.id = pt.post_id AND p.status = 2
GROUP BY t.id, t.slug, t.name
ORDER BY post_count DESC, t.name;

-- name: GetTagBySlug :one
SELECT id, slug, name
FROM tags
WHERE slug = ?
LIMIT 1;

-- name: ListPublishedPostsByTag :many
SELECT p.id, p.public_id, p.slug, p.title, p.summary, p.cover_url,
       p.reading_time_min, p.published_at, p.author_id
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE p.status = 2 AND t.slug = ?
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedPostsByTag :one
SELECT COUNT(*) AS total
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE p.status = 2 AND t.slug = ?;

-- name: SearchPublishedPosts :many
-- ngram 全文检索：中文按 2 字分词，最短需 2 字才有结果。
-- 用 NATURAL LANGUAGE MODE 拿相关性分排序；relevance > 0 即命中。
SELECT id, public_id, slug, title, summary, cover_url, reading_time_min, published_at, author_id,
       MATCH(title, content_md) AGAINST (? IN NATURAL LANGUAGE MODE) AS relevance
FROM posts
WHERE status = 2
  AND MATCH(title, content_md) AGAINST (? IN NATURAL LANGUAGE MODE)
ORDER BY relevance DESC, published_at DESC
LIMIT ? OFFSET ?;

-- name: CountSearchPublishedPosts :one
SELECT COUNT(*) AS total
FROM posts
WHERE status = 2
  AND MATCH(title, content_md) AGAINST (? IN NATURAL LANGUAGE MODE);

-- name: ListArchiveMonths :many
SELECT DATE_FORMAT(published_at, '%Y-%m') AS ym, COUNT(*) AS total
FROM posts
WHERE status = 2 AND published_at IS NOT NULL
GROUP BY ym
ORDER BY ym DESC;

-- name: ListPublishedPostsByMonth :many
-- 用半开区间而不是 DATE_FORMAT 字符串比较，才能命中 (status, published_at) 索引
SELECT id, public_id, slug, title, summary, cover_url, reading_time_min, published_at, author_id
FROM posts
WHERE status = 2
  AND published_at >= ?
  AND published_at < ?
ORDER BY published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedPostsByMonth :one
SELECT COUNT(*) AS total
FROM posts
WHERE status = 2
  AND published_at >= ?
  AND published_at < ?;

-- name: ListPublishedPostsByAuthor :many
-- 某用户的公开博客页（/users/{public_id}）：只列已发布，按对外 public_id 找人。
SELECT p.id, p.public_id, p.slug, p.title, p.summary, p.cover_url,
       p.reading_time_min, p.published_at, p.author_id
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.status = 2 AND u.public_id = ?
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedPostsByAuthor :one
SELECT COUNT(*) AS total
FROM posts p
JOIN users u ON u.id = p.author_id
WHERE p.status = 2 AND u.public_id = ?;

-- name: GetAuthorPublishStats :one
-- 「我的视角」统计（首页 Hero 与侧边栏个人卡片用）：本人已发布文章的
-- 文章数 / 涉及的去重标签数 / 写作月数。注意 post_total 必须用
-- COUNT(DISTINCT p.id)——JOIN post_tags 会把一篇文章翻成多行，COUNT(*) 会虚高。
SELECT
  COUNT(DISTINCT p.id) AS post_total,
  COUNT(DISTINCT pt.tag_id) AS tag_total,
  COUNT(DISTINCT DATE_FORMAT(p.published_at, '%Y-%m')) AS month_total
FROM posts p
LEFT JOIN post_tags pt ON pt.post_id = p.id
WHERE p.status = 2 AND p.author_id = ?;
