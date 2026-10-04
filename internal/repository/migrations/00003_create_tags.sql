-- +goose Up
-- +goose NO TRANSACTION
CREATE TABLE tags (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  slug       VARCHAR(100) NOT NULL,
  name       VARCHAR(100) NOT NULL,
  post_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '冗余计数，service 层事务内维护',
  PRIMARY KEY (id),
  UNIQUE KEY uq_tags_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='标签';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS tags;
