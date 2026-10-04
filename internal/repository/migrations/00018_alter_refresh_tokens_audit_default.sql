-- refresh_tokens 只有 created_at（无 updated_at），理由见 00014。
--
-- expires_at / revoked_at 是业务字段，不加默认值。

-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE refresh_tokens
  MODIFY created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3);

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE refresh_tokens
  MODIFY created_at DATETIME(3) NOT NULL;
