-- 媒体库查询。图片文件存本地磁盘，这里只记元数据。
-- 归属：author_id=0 是历史遗留（仅管理员可见），其余只有上传者本人与管理员可见。

-- name: CreateMedia :execlastid
INSERT INTO media (public_id, url, filename, size_bytes, mime_type, width, height, author_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListMedia :many
-- author_filter=0 表示不过滤（管理员），否则只看 author_id 匹配的行。
SELECT id, public_id, url, filename, size_bytes, mime_type, width, height, created_at
FROM media
WHERE (? = 0 OR author_id = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountMedia :one
SELECT COUNT(*) AS total
FROM media
WHERE (? = 0 OR author_id = ?);

-- name: SumMediaSize :one
-- 配额校验用库里累计的体积，不每次扫磁盘（全站配额，按人配额见 devlog 遗留）。
-- CAST 成 SIGNED 让 sqlc 生成 int64（UNSIGNED 会退化成 interface{}）。
SELECT CAST(COALESCE(SUM(size_bytes), 0) AS SIGNED) AS total
FROM media;

-- name: GetMediaByPublicID :one
SELECT id, public_id, url, filename, size_bytes, mime_type, width, height, author_id, created_at
FROM media
WHERE public_id = ?
LIMIT 1;

-- name: DeleteMediaByPublicID :exec
DELETE FROM media
WHERE public_id = ?;
