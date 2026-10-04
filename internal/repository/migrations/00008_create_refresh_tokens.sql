-- +goose Up
-- +goose NO TRANSACTION
-- token_hash 只存哈希；检测到已撤销 token 被重放时，撤销该用户全部会话
CREATE TABLE refresh_tokens (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id    BIGINT UNSIGNED NOT NULL,
  token_hash VARCHAR(64) NOT NULL,
  expires_at DATETIME(3) NOT NULL,
  revoked_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_refresh_token_hash (token_hash),
  KEY idx_refresh_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='刷新令牌';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS refresh_tokens;
