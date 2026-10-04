-- 新增「站点管理员」标识，与 can_publish（能否发帖）解耦。
--
-- 个人博客的权限模型分两层：
--   1. can_publish=1  → 可进入「我的博客」编辑管理界面，发布/管理自己的文章、媒体、评论。
--   2. is_admin=1     → 额外可进入「纯管理后台」，管理用户与权限（用户管理页）。
-- 普通作者（只有 can_publish）看不到也进不了后台；只有站点管理员能进后台。
-- 这样「个人博客编辑界面」与「后台管理」就是两套独立的东西。

-- +goose Up
ALTER TABLE users ADD COLUMN is_admin TINYINT NOT NULL DEFAULT 0;

-- 把已有的种子站长标记为该站管理员（邮箱见技术方案 M2 补充）
UPDATE users SET is_admin = 1 WHERE email = 'author@seed.local';

-- +goose Down
ALTER TABLE users DROP COLUMN is_admin;
