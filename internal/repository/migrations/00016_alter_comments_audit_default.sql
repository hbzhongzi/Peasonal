-- comments 只有 created_at（无 updated_at），理由见 00014。

-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE comments
  MODIFY created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3);

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE comments
  MODIFY created_at DATETIME(3) NOT NULL;
