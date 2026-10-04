-- +goose Up
-- +goose NO TRANSACTION
-- 目录（TOC）是渲染管线的产物之一，必须和 content_html 一起落库。
--
-- 为什么不在详情页实时解析 content_html 生成目录：
--   ① 实时解析等于每次请求都要跑一遍 DOM 解析，白扔 CPU；
--   ② TOC 与正文同属一次渲染的产物，一起写库才能保证两者永不脱节。
--
-- 用 TEXT 而不是 JSON 类型：sqlc 对 MySQL 的 JSON 列类型推断不稳定，
-- 而 TOC 只有几 KB，TEXT（64KB 上限）绰绰有余，取出来直接 json.Unmarshal 即可。
ALTER TABLE posts ADD COLUMN toc_json TEXT NULL COMMENT '目录 JSON，渲染时生成' AFTER content_html;

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE posts DROP COLUMN toc_json;
