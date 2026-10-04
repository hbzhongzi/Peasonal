-- posts 审计字段补齐默认值，理由见 00014。
--
-- 注意 published_at 是业务字段（可空，草稿无发布时间），不是审计字段，不加默认值。

-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE posts
  MODIFY created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  MODIFY updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3);

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE posts
  MODIFY created_at DATETIME(3) NOT NULL,
  MODIFY updated_at DATETIME(3) NOT NULL;
