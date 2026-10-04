-- +goose Up
-- +goose NO TRANSACTION
-- content_md 存 Markdown 原文，content_html 存渲染结果（见设计文档 §2.3）
-- 用 MEDIUMTEXT 而非 TEXT：TEXT 上限仅 64KB，长文内联高亮样式很容易超
CREATE TABLE posts (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id        VARCHAR(26)  NOT NULL COMMENT 'ULID，对外标识',
  slug             VARCHAR(200) NOT NULL COMMENT 'URL 友好标识',
  title            VARCHAR(255) NOT NULL,
  summary          VARCHAR(500) NOT NULL DEFAULT '' COMMENT '列表摘要，为空则自动截取',
  content_md       MEDIUMTEXT   NOT NULL,
  content_html     MEDIUMTEXT   NOT NULL,
  cover_url        VARCHAR(512) NOT NULL DEFAULT '',
  status           TINYINT      NOT NULL DEFAULT 1 COMMENT '1草稿 2已发布 3归档',
  comment_enabled  TINYINT(1)   NOT NULL DEFAULT 1,
  reading_time_min INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '渲染时计算',
  published_at     DATETIME(3)  NULL COMMENT '草稿为 NULL',
  created_at       DATETIME(3)  NOT NULL,
  updated_at       DATETIME(3)  NOT NULL,
  author_id        BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_posts_public_id (public_id),
  UNIQUE KEY uq_posts_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文章';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS posts;
