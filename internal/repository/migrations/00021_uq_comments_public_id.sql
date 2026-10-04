-- +goose Up
-- +goose NO TRANSACTION
CREATE UNIQUE INDEX uq_comments_public_id ON comments (public_id);

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX uq_comments_public_id ON comments;
