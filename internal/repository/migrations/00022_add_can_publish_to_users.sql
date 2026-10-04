-- +goose Up
-- +goose NO TRANSACTION
-- 新增「是否可发布文章」标志。
-- 个人博客默认只有博主（管理员）能发帖；公开注册的用户默认可评论/上传但不可发帖，
-- 发帖权限由管理员在 /admin/users 另行开启（见开发规划 M2 补充）。
ALTER TABLE users
  ADD COLUMN can_publish TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否允许发布文章，由管理员开启';

-- 存量数据：原本能上传的管理员一并赋予发布权限，保证现有账号不丢权限
UPDATE users SET can_publish = 1 WHERE can_upload = 1;

-- +goose Down
-- +goose NO TRANSACTION
ALTER TABLE users DROP COLUMN can_publish;
