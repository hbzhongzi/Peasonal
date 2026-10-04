-- 删除 tags.post_count 冗余列。
--
-- 背景：标签计数实际由 ListTagsWithCount 用 JOIN + COUNT 实时算出
-- （只统计已发布文章，草稿挂着的标签不该出现在前台）。
-- 而 tags.post_count 这个物理列自建表以来从未被任何代码写入，永远是 0。
--
-- 为什么删列而不是补维护逻辑：
--   1. 维护它的成本不低 —— 文章发布/转草稿/删除、标签增删都要重算，漏一处就不一致；
--   2. 它作为"缓存"的意义在本项目规模下不成立（<1000 篇文章，JOIN COUNT 代价可忽略）；
--   3. 最危险的是它**看起来是对的**：列在那、类型对、名字合理，
--      将来谁写一句 SELECT post_count FROM tags 会拿到 0 而不起疑。
--
-- 宁可没有这个字段，也不要一个永远错误却看似正常的字段。

-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE tags DROP COLUMN post_count;

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE tags ADD COLUMN post_count INT UNSIGNED NOT NULL DEFAULT 0;
