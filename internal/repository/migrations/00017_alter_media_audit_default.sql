-- media 只有 created_at（无 updated_at），理由见 00014。
--
-- 改这列会重建 idx_media_created 索引，但表是空的，代价可忽略。

-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE media
  MODIFY created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3);

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE media
  MODIFY created_at DATETIME(3) NOT NULL;
