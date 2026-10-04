-- +goose Up
-- +goose NO TRANSACTION
-- 本期无审核流程：提交即写入 status=1 直接公开（见设计文档 §7.5）
-- status 字段保留，将来 spam 变多时改成审核只需改写入默认值，不必改表
CREATE TABLE comments (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  post_id      BIGINT UNSIGNED NOT NULL,
  author_name  VARCHAR(64)  NOT NULL,
  author_email VARCHAR(191) NOT NULL DEFAULT '' COMMENT '不对外展示',
  content      TEXT         NOT NULL,
  status       TINYINT      NOT NULL DEFAULT 1 COMMENT '1公开 2已隐藏',
  ip_hash      VARCHAR(64)  NOT NULL DEFAULT '' COMMENT '隐私合规，只存加盐哈希',
  created_at   DATETIME(3)  NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='评论';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS comments;
