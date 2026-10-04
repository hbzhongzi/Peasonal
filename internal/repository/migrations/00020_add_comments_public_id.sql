-- +goose Up
-- +goose NO TRANSACTION
-- 评论对外暴露 public_id（ULID），与 posts 保持一致：不暴露自增 id，避免被遍历。
ALTER TABLE comments ADD COLUMN public_id VARCHAR(26) NOT NULL DEFAULT '' AFTER id;

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE comments DROP COLUMN public_id;
