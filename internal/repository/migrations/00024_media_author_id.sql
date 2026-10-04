-- 媒体库按用户隔离：每张图片记录上传者。
--
-- 隐私规则（见 devlog M2 补充四）：
--   author_id = 0  表示历史遗留（加列前上传的），只有管理员能看见、能管理；
--   author_id > 0  只有上传者本人和管理员能看见、能管理。
-- 不回填历史数据，避免把别人的图算到某个人头上。

-- +goose Up
ALTER TABLE media
  ADD COLUMN author_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD KEY idx_media_author (author_id);

-- +goose Down
ALTER TABLE media
  DROP KEY idx_media_author,
  DROP COLUMN author_id;
