-- +goose Up
-- +goose NO TRANSACTION
CREATE TABLE users (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id     VARCHAR(26)  NOT NULL COMMENT 'ULID，对外标识',
  email         VARCHAR(191) NOT NULL COMMENT 'utf8mb4_0900_ai_ci 天然不区分大小写',
  password_hash VARCHAR(255) NOT NULL COMMENT 'argon2id',
  display_name  VARCHAR(64)  NOT NULL DEFAULT '',
  can_upload    TINYINT(1)   NOT NULL DEFAULT 1 COMMENT '是否允许上传图片，见设计文档 §5.5',
  created_at    DATETIME(3)  NOT NULL,
  updated_at    DATETIME(3)  NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_users_public_id (public_id),
  UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='管理员';

-- +goose Down
-- +goose NO TRANSACTION
DROP TABLE IF EXISTS users;
