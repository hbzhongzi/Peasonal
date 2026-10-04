-- +goose Up
-- +goose NO TRANSACTION
CREATE TABLE post_views (
  post_id   BIGINT UNSIGNED NOT NULL,
  view_date DATE            NOT NULL,
  count     INT UNSIGNED    NOT NULL DEFAULT 0,
  PRIMARY KEY (post_id, view_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='每日阅读量';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS post_views;
