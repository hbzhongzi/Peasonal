-- name: GetUserByEmail :one
SELECT id, public_id, email, password_hash, display_name, can_upload, can_publish, is_admin,
       bio, avatar_url, github, created_at, updated_at
FROM users
WHERE email = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT id, public_id, email, password_hash, display_name, can_upload, can_publish, is_admin,
       bio, avatar_url, github, created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1;

-- name: GetUserByPublicID :one
SELECT id, public_id, email, password_hash, display_name, can_upload, can_publish, is_admin,
       bio, avatar_url, github, created_at, updated_at
FROM users
WHERE public_id = ?
LIMIT 1;

-- name: GetFirstAdminProfile :one
-- 站点公开资料：取第一个管理员（即站长）。前台首页博主卡片从这里取值。
SELECT public_id, email, display_name, bio, avatar_url, github
FROM users
WHERE is_admin = 1
ORDER BY id ASC
LIMIT 1;

-- name: UpdateMyProfile :exec
-- 用户改自己的公开资料（昵称/简介/头像/GitHub），id 取自 access token，不接受路径参数。
UPDATE users
SET display_name = ?, bio = ?, avatar_url = ?, github = ?, updated_at = NOW()
WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) AS total
FROM users;

-- name: ListUsers :many
SELECT id, public_id, email, display_name, can_upload, can_publish, is_admin, created_at
FROM users
ORDER BY created_at ASC, id ASC;

-- name: UpdateUserPermissions :exec
UPDATE users
SET can_upload = ?, can_publish = ?, is_admin = ?, updated_at = NOW()
WHERE public_id = ?;

-- name: GetPublicUserByID :one
-- 公开资料（按内部 id）：文章详情页显示作者用。刻意不含邮箱/密码/权限位。
SELECT public_id, display_name, bio, avatar_url, github
FROM users
WHERE id = ?
LIMIT 1;

-- name: ListPublicUsersByIDs :many
-- 批量公开资料：文章列表页按 author_id 批量补作者，避免 N+1。
SELECT id, public_id, display_name, avatar_url
FROM users
WHERE id IN (sqlc.slice('ids'));

-- name: GetPublicUserByPublicID :one
-- 公开资料（按对外 id）：/users/{public_id} 个人博客主页用。
SELECT public_id, display_name, bio, avatar_url, github
FROM users
WHERE public_id = ?
LIMIT 1;
