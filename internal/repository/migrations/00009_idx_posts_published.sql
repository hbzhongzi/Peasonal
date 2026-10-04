-- +goose Up
-- +goose NO TRANSACTION
-- MySQL 无部分索引，改用复合索引把 status 放在最左列（见设计文档 §1.2.1）
CREATE INDEX idx_posts_published ON posts (status, published_at DESC);

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX idx_posts_published ON posts;
