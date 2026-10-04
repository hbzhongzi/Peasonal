-- 审计字段补齐默认值。
--
-- 背景：所有表的 created_at / updated_at 都定义为 DATETIME(3) NOT NULL 但无默认值。
-- MySQL 严格模式下，INSERT 少传一个就会报 Error 1364 "doesn't have a default value"。
-- 这不是某条 SQL 写漏了，而是 schema 层没有兜底 —— 靠每个调用点自觉传值必然再犯。
--
-- 决策：让数据库兜底，而不是补 SQL。
--   - created_at 默认 CURRENT_TIMESTAMP(3)：插入不传即自动填
--   - updated_at 额外 ON UPDATE CURRENT_TIMESTAMP(3)：任何 UPDATE 自动维护，
--     省掉每条 UPDATE 都要记得写 updated_at = ?（漏写是这类字段最常见的 bug）
--
-- 为什么改迁移而不是改 00001：00001 已执行过，改写已执行的迁移是反模式；
-- 一旦存在第二个环境，两边的 goose_version 就会对不上。这里追加 ALTER 是唯一安全做法。

-- +goose Up
-- +goose NO TRANSACTION
ALTER TABLE users
  MODIFY created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  MODIFY updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3);

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE users
  MODIFY created_at DATETIME(3) NOT NULL,
  MODIFY updated_at DATETIME(3) NOT NULL;
