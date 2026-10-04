-- 刷新令牌查询。refresh token 本身是随机串，库里只存其 SHA-256。
-- 这样库被读走也无法冒用，且能按哈希精确撤销。

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at)
VALUES (?, ?, ?, ?);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = ?
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = ?
WHERE token_hash = ?;

-- name: RevokeUserTokens :exec
-- 登出「所有设备」：撤销该用户全部未过期令牌
UPDATE refresh_tokens
SET revoked_at = ?
WHERE user_id = ? AND revoked_at IS NULL;
