-- +goose Up
-- +goose NO TRANSACTION
-- 主键是 (post_id, tag_id)，按 tag_id 反查需要单独建索引
CREATE INDEX idx_post_tags_tag ON post_tags (tag_id, post_id);

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX idx_post_tags_tag ON post_tags;
