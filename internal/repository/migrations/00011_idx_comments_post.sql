-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX idx_comments_post ON comments (post_id, status, created_at);

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX idx_comments_post ON comments;
