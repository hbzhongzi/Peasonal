# M3 · 评论与搜索

## 背景

M3 目标是让读者能留言、管理员能清理垃圾、访客能搜到文章。评论与搜索都是公开侧能力，但管理入口放在 `/admin`。

## 1. 评论基础能力

- `comments` 表结构：公开 ID、文章 ID、昵称、邮箱、内容、状态、IP 哈希、创建时间。
- 评论提交：无审核，`status=1` 直接公开。
- 防垃圾四道闸：
  1. 蜜罐字段：机器人填了静默丢弃，返回成功假象。
  2. 最小间隔：同一 IP 10 秒内不能连发。
  3. 频率：同一 IP 每分钟最多 1 条。
  4. 日上限：同一 IP 每天最多 50 条。
- IP 只存加盐哈希，不存明文。
- 后台评论管理：`/admin/comments` 用 `RequireAdmin`，可隐藏/删除。

## 2. 搜索能力

- 基于 MySQL ngram 全文检索：`ft_posts_search` 索引，`ngram_token_size=2`。
- 使用 `MATCH(title, content_md) AGAINST (? IN NATURAL LANGUAGE MODE)` 按相关性排序。
- 中文 2 字起搜；数据量小用自然语言模式足够。

## 3. 踩坑

- 无审核导致垃圾评论上墙：防垃圾前置到提交时，`status` 字段保留，被刷时改默认值即可切审核，不用改表。
- 阅读量统计未做，详情页暂不显示阅读数。

## 4. 初始实现

- 后端：`GET /posts/{slug}/comments` 一次性返回全部公开评论；`POST /posts/{slug}/comments` 提交；`CommentService.Create` 实现四道闸。
- 前端：`CommentSection` 默认展示评论列表 + 提交表单。

## 5. 补充（2026-09-30）：公开评论列表分页 + 折叠图标

用户要求"文章下方加评论图标，点击查看历史评论，支持滑动分页加载"。原 `GET /posts/{slug}/comments` 一次性返回全部公开评论，不满足滚动加载；本次补齐：

- **后端**：`comments.sql` 新增 `ListPublicCommentsByPostPaginated`（LIMIT/OFFSET）；`CommentService` 新增 `ListPublicPaginated`（返回带 `total` 的 `domain.Page[Comment]`，复用 `CountCommentsByPost`）+ `CountPublic`；`comment_handler` 的公开列表接口支持 `page/page_size` 并返回分页结构，新增 `GET /posts/{slug}/comments/count` 仅返回公开评论总数。
- **前端**：`commentsApi.listPublic` 改为分页、返回 `Page<Comment>`；新增 `count`；新增 `features/comments/hooks.ts`（`useComments` 用 `useInfiniteQuery` 滚动分页 + `useCommentCount`）；`CommentSection` 重写为默认折叠——只显示「💬 评论（数量）」图标按钮，展开后用 `IntersectionObserver` 触底加载下一页。
- 验证：`go build`/`npm run build`（122 模块）通过；curl 实测 count=0 → 发帖后 1、分页列表正确返回。

## 6. 补充（2026-09-30）：评论昵称跟随用户

用户反馈：**「评论接口有点问题并且昵称应该跟随用户的昵称不支持手动更改」**。处理：

- `web/src/features/comments/components/CommentSection.tsx`：
  - 已登录用户：通过 `useAuth` 读取 `display_name`（无昵称时用邮箱前缀兜底），自动作为 `author_name` 提交；表单内隐藏昵称输入框，改为显示「以 XXX 的身份发表评论」提示。
  - 匿名用户：仍保留「昵称（可选）」输入框，留空时默认使用「匿名」。
- 后端 `CommentService.Create` 逻辑不变，继续校验昵称非空且长度 ≤64；防垃圾四道闸照常生效。

验证：`npm run build` 124 模块（JS gzip 141.46 KB）通过。

说明：截图中的「评论太快了，请稍后再试」是 IP 限流的 10 秒最小间隔正常生效，不是接口 bug。本地开发时所有请求共用一个 `127.0.0.1`，所以同一机器上连续测试会触发；生产环境配合 `X-Forwarded-For` 会按真实客户端 IP 限流。
