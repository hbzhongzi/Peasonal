-- +goose Up
-- +goose NO TRANSACTION
CREATE TABLE post_tags (
  post_id BIGINT UNSIGNED NOT NULL,
  tag_id  BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (post_id, tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文章与标签关联';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS post_tags;
