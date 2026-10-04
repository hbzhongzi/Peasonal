-- 用户个人资料字段：博主身份从「前端常量 site.ts」挪进数据库。
--
-- 每个用户有自己的公开身份（昵称/一句话简介/头像/GitHub），
-- 在「我的博客 → 个人资料」里自行维护；前台首页的博主卡片
-- 读站点管理员（is_admin=1 的第一个用户）的公开资料。
-- email 列已有，不再重复。

-- +goose Up
ALTER TABLE users
  ADD COLUMN bio VARCHAR(512) NOT NULL DEFAULT '' COMMENT '一句话简介（Hero/侧边栏）',
  ADD COLUMN avatar_url VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像图片地址，空则用首字母占位',
  ADD COLUMN github VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'GitHub 主页链接';

-- +goose Down
ALTER TABLE users
  DROP COLUMN bio,
  DROP COLUMN avatar_url,
  DROP COLUMN github;
