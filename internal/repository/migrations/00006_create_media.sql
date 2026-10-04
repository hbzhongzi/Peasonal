-- +goose Up
-- +goose NO TRANSACTION
-- 图片存本地磁盘，这里只记元数据（见设计文档 §7.2）
-- width/height 必须记录，供前端占位防 CLS
CREATE TABLE media (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id  VARCHAR(26)  NOT NULL,
  url        VARCHAR(512) NOT NULL,
  filename   VARCHAR(255) NOT NULL COMMENT '磁盘上的文件名 <ULID>.webp',
  size_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
  mime_type  VARCHAR(100) NOT NULL DEFAULT '',
  width      INT UNSIGNED NOT NULL DEFAULT 0,
  height     INT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3)  NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_media_public_id (public_id),
  KEY idx_media_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='媒体文件';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS media;
