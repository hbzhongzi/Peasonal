-- +goose Up
-- +goose NO TRANSACTION
-- ngram parser：MySQL 8.0 内置，中文全文检索开箱即用
--
-- 两个硬性注意点（见设计文档 §4.2 / §7.3）：
--   1. 查询必须用 IN BOOLEAN MODE。NATURAL LANGUAGE MODE 有 50% 阈值：
--      词出现在超过一半的行里会被直接忽略，文章少时这个坑极其明显。
--   2. ngram_token_size 是只读启动参数（默认 2），改了必须重启 MySQL
--      并 DROP + CREATE 重建本索引才生效。
CREATE FULLTEXT INDEX ft_posts_search ON posts (title, content_md) WITH PARSER ngram;

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX ft_posts_search ON posts;
