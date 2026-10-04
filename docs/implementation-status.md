# 个人博客系统 · 开发落地总览

> **这是"项目现在到哪了"的唯一入口。** 想知道当前进度、能跑什么、怎么跑、改过什么，只看这一个文件。
>
> 最后更新：2026-10-04 ｜ 当前阶段：**M1/M2 已完成并端到端验证 ✅；M3 评论与搜索已完成（评论公开列表支持分页 + 滚动加载），M4 待启动；**M5 Docker 线上部署手册与配置已产出（`docs/docker-production-deploy.md`、根目录 Dockerfile/Compose/Caddyfile，尚未在真实服务器演练）**（M2 追加补充（八）：普通用户默认即作者、上传权限改由管理员授予；补充（九）：首页移除 Hero、关于页标题按视角显示、工作台移除「前台」按钮、评论折叠图标 + 滚动分页；补充（十）：工作台导航移除「媒体库」、增加「首页」返回前台入口；补充（十一）：后台顶栏移除「我的博客」「前台」跳转按钮；补充（十二）：用户只能编辑自己的博客 + 后台不再跳用户主页；补充（十三）：用户菜单「我的主页」改为回首页；补充（十四）：管理员独立登录页 /admin/login；补充（十五）：工作台顶栏改为用户菜单下拉）

---

## 1. 当前进度

```
M0  工程地基          ██████████  5/5    ✅ 完成
M1  文章核心闭环      ██████████  21/21 ✅ 完成并端到端验证
M2  管理后台          ██████████  17/17 ✅ 完成并端到端验证
M3  评论与搜索        █████████░  6/7   ✅ 完成并端到端验证（阅读量统计未做）
M4  SEO 与内容分发    ░░░░░░░░░░  0/5   ⬜ 待启动
M5  容器化与上线      ░░░░░░░░░░  0/6   ⏸ 暂缓（用户明确：先本地落地）
```

| 阶段 | 状态 | 完成度 | 落地记录 |
| --- | --- | --- | --- |
| 2026-09-29 | M2+M3 补 | 修复从子目录 `go run ./cmd/server` 启动找不到 `.env` 导致 `DATABASE_URL` 必填失败：config.Load 改为从 cwd 逐级向上查找 `.env`，并把 MEDIA_DIR 等相对路径锚定到仓库根，uploads 不再落到子目录 |
| M0 工程地基 | ✅ 完成 | 5/5 | [devlog/M0-foundation.md](./devlog/M0-foundation.md) |
| M1 文章核心闭环 | ✅ 完成并验证 | 21/21 | [devlog/M1-content-loop.md](./devlog/M1-content-loop.md) |
| M2 管理后台 | ✅ 完成并验证 | 17/17 | [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) |
| M3 评论与搜索 | ✅ 完成并验证（阅读量待补） | 6/7 | [devlog/M3-comments-search.md](./devlog/M3-comments-search.md) |
| M4 SEO 与内容分发 | ⬜ 待启动 | 0/5 | — |
| M5 容器化与上线 | ⏸ 暂缓 | 0/6 | — |

---

## 2. 当前可用能力

> 诚实清单 —— 只列**已经验证过**的能力。

### ✅ 已验证（有测试或构建覆盖）

| 能力 | 验证方式 |
| --- | --- |
| Markdown 渲染管线 | 10 个单测全过（代码块 class-based 高亮、GFM 表格、任务列表、中文 anchor 去重、外链属性、图片属性、XSS 拦截、LRU 缓存、**h1 不进摘要与目录**、**行内代码保留·块级代码跳过**） |
| 摘要生成 | 单测 + `make seed` 真实入库后核对 |
| slug 生成 | 3 个单测（含中文、长度上限、冲突加序号） |
| 后端编译 / 静态检查 | `go build ./...` + `go vet ./...` |
| 前端类型检查 / 构建 | `npm run build`（tsc + vite） |
| 服务启动与健康检查 | `/healthz`、`/readyz`、`/api/v1/ping` 均 200 |
| 数据库迁移 | **25 个全部 up 成功**（含 M2 的 00020/00021、00022 can_publish、00023 is_admin、00024 media.author_id、00025 用户资料三列）（含 ngram FULLTEXT、`toc_json`、审计字段默认值、删 `post_count`） |
| 文章列表 / 详情 / 标签 / 归档 API | 6 个接口全部 curl 实测，返回正确 |
| 错误路径 | 不存在的 slug → 404 + 统一错误体；分页越界 → 空列表不报错 |
| `cmd/seed` 导入与幂等 | 灌 3 篇成功；连跑两次 posts 仍为 3，未重复 |
| 前后端字段契约 | 逐字段核对一致（`public_id`/`content_html`/`toc`/`comment_enabled` 等） |
| 前端 → 后端代理 | `localhost:5173/api/v1/*` 正常透传到 `:8080` |
| 认证与授权 | 登录拿 access token + HttpOnly refresh cookie；`/admin/me` 校验；未登录访问 `/admin/*` → 401；refresh 轮换 | 
| 文章写路径 | 建/改/发/下线/删 + 评论开关；保存时触发渲染管线，`content_html`+`toc_json` 落库；slug 唯一；标签幂等重链 |
| 图片上传（本地） | `MaxBytesReader` 读前截断（超限 413 且不进内存）；MIME 读文件头、禁 SVG；GIF 直出；超 1600px 缩放；配额拒绝；`/uploads/*` 公开静态直出 |
| 评论 | 提交即公开（无审核）；四道防垃圾闸（蜜罐假象/10s 间隔/1 条每分/50 每日本地 IP）；后台隐藏/恢复/删除；**公开列表分页**（`GET /posts/{slug}/comments?page=&page_size=` 带 `total`）+ **计数端点**（`GET /posts/{slug}/comments/count`，评论图标角标用）；前端 `CommentSection` 默认折叠为「💬 评论（数量）」图标，展开后 `IntersectionObserver` 触底滚动加载下一页；**已登录用户评论昵称自动取自 display_name，不可手动修改，匿名用户仍可自定义昵称** |
| 搜索 | ngram FULLTEXT 中文+英文命中，`/posts/search` 实测通过 |
| 注册（公开） | `POST /admin/register` 创建普通用户并自动登录（注册即登录），邮箱去重（重复 409）、密码 ≥8 位 |
| 权限分层（三级） | `is_admin` 控制进入纯管理后台、`can_publish` 控制发帖与管理自己的博客、`can_upload` 控制上传；中间件 `RequireAdmin`/`RequirePublish`/`RequireUpload` 三级隔离；**普通注册用户默认 `can_publish=1, can_upload=0, is_admin=0`**（注册即作者，上传为管理员授予的特权，默认关闭），发帖/上传权限由管理员在后台开关 |
| 用户管理（纯后台） | `GET/PATCH /admin/users` 由 `RequireAdmin` 保护，仅站点管理员可进；列出用户并开关发帖/上传/**管理员**权限；前端 `/admin`（后台管理）页面可逐用户开关，含「管理员」开关 |
| **数据按用户隔离（M2 补充十二）** | 文章与媒体一律按 `author_id` 过滤：每个人（包括站点管理员）在「我的博客」工作台只能看/改/删自己的；跨用户写操作 → 403「只能操作自己创建的内容」。统一由 `service.Actor`（`AuthorFilter()` / `CanTouch()`）判定，查询用 `WHERE (? = 0 OR author_id = ?)` 让 sqlc 静态分析。管理员不再能通过 `/studio` 查看全站文章/媒体 |
| **个人资料（本人可改）** | `PATCH /admin/me` 改昵称/简介/头像/GitHub（id 只取自 token，不接受路径参数）；校验昵称非空、简介 ≤512、头像只收 `/` 站内路径或 `https://`、GitHub 只收 `http(s)://`（挡 `javascript:`）；前端 `/studio/profile` 页面，**对所有登录用户开放（与发帖权限无关，M2 补充六）** |
| **站长公开资料（匿名）** | `GET /site/profile` 返回站长（id 最小的管理员）的昵称/简介/头像/GitHub，**不含邮箱**；前端 侧边栏/关于页/页脚改由它驱动（关于页标题按视角：站长本人看「关于我」、其他用户/匿名看「关于作者」），`config/site.ts` 退化为兜底默认值 |
| **前台与后台断链（M2 补充七 + 十一）** | 前台、工作台、后台**任何位置都没有**跨界面跳转按钮：① 前台无进入 `/admin`/`/studio` 的入口（导航/用户菜单/详情页「编辑」/登录落地分流均已移除）；② 工作台无进入 `/admin` 的入口，仅保留「首页」返回前台；③ 后台无进入 `/studio` 或前台的入口，`/admin` 与 `/studio` 均只认 URL 直访（未登录踢到登录页，登录后**回来源页**） |
| **作者归属与用户博客主页** | 公开列表/详情每篇带 `author{public_id,display_name,avatar_url}`（`attachAuthors` 批量补全，无 N+1）；卡片与详情页署名并链接 `/blog/{public_id}`；匿名接口 `GET /users/{publicID}`（公开资料）+ `GET /users/{publicID}/posts`（只列已发布）；`/blog/:publicID` 用户主页；**顶栏用户菜单**（匿名 → 登录/注册；登录 → 头像+昵称展开：我的主页 → `/`、个人资料 → `/studio/profile`、退出）。curl 实测逐篇署名正确、404 路径正确 |
| **普通用户默认即作者 + 上传权限管理员授予（M2 补充八）** | 注册默认 `can_publish=1`（注册即为作者，可在「我的博客」写/发文章）、`can_upload=0`（上传为管理员授予的特权，默认关闭）、`is_admin=0`；**本人查看自己的 `/blog/{publicID}` 主页**时，资料头下方出现「写新文章 / 管理我的博客」入口（仅对自己可见，是内容操作而非后台入口）；媒体库前端按 `can_upload` 显隐上传按钮，无权限显示「无上传权限」提示；后端 `RequireUpload` 仍兜底。现有普通用户（爱丽丝/鲍勃/joker/12/卡罗）均已置 `can_publish=1`。curl 实测：新注册用户 `can_publish=true, can_upload=false`、可进 `/studio` 写文章接口；`can_upload=0` 的令牌上传返回 403、`can_upload=1` 返回 201 |
| **工作台导航（M2 补充十 + 十五）** | `StudioLayout` 顶栏导航改为 **首页 / 文章（按权限）**；移除「媒体库」入口（`/studio/media` 路由仍保留，用户可直接输 URL 访问）；保留「首页」作为返回前台的唯一出口；「个人资料 / 退出登录」收到右上角用户菜单下拉中，与前台 UserMenu 样式一致 |
| **后台导航（M2 补充十一）** | `AdminLayout` 顶栏仅保留 **用户 / 评论**；移除「我的博客」「前台」跳转按钮；无权限提示页同样移除「返回博客首页」按钮，只保留「退出登录」 |
| **后台不跳用户主页（M2 补充十二）** | 用户管理页（`UsersAdminPage`）移除每行「博客」按钮，管理员不再能从后台直接跳转 `/blog/{public_id}`；用户主页仍公开可访，但只能靠前台/URL 直访 |
| **管理员独立登录页（M2 补充十四）** | 新增 `/admin/login` 页面，作为管理员专属登录入口；未登录访问 `/admin/*` 时重定向到 `/admin/login`；登录后前端校验 `is_admin=true`，非管理员即使密码正确也会被拒绝并强制登出；普通用户继续使用 `/login`。`npm run build` 124 模块 |
| **注册校验** | 邮箱正则要求带「.后缀」（`12@123`/`a@b` → 400「邮箱格式不正确」）；密码 ≥8 位；昵称留空取邮箱前缀；注册即登录（M2 补充六） |
| 后台与个人博客界面分离 | 「我的博客」(`/studio`，`can_publish` 可进) 负责文章/媒体/评论等个人编辑；「后台」(`/admin`，`is_admin` 可进) 仅负责用户/权限管理。两套独立界面，普通作者进得了工作台、进不了后台 |
| 前端 401 自动续期 | 任意请求 401 时用 refresh cookie 换新会话并重试原请求（并发去重）；刷新失败回登录态；`coreFetch` 重试用独立超时控制器，避免原请求超时导致重试也被 abort |
| 登录/注册落地分流 | 登录后**不按权限分流**：有来源页（从 /studio、/admin 被踢出）→ 回来源页；无来源 → 回首页。注册即登录后回博客首页。后台入口只有 URL 直访这一条路（M2 补充七） |
| 编辑器保存反馈 | 保存成功后显示「已于 hh:mm 保存」；草稿/发布成功后同步表单状态（按钮文案随发布状态切换）；错误横幅即时清空 |
| 前端类型检查 / 构建 | `npm run build`（tsc + vite）✅ 通过，**125 模块** |

### ⏸ 已实现但未运行时验证

| 能力 | 原因 |
| --- | --- |
| 主题三态切换 + 首屏无闪烁 | 需浏览器实机确认（构建通过、CSS 已生成双主题） |
| 首页文章列表 / 侧边栏 / 关于页标题视角的视觉效果 | 接口数据已接通、构建通过，具体观感待用户确认 |
| 顶栏用户菜单的下拉开合（点外部/Esc 关闭） | 逻辑已实现，交互手感待浏览器实机确认 |

### ⬜ 尚未开始

SEO（M4）、容器化与上线（M5，用户明确暂缓）。

> M2/M3 的"自动续期拦截"（401 时自动 refresh 并重试原请求）已由 store 提供 `bootstrap` 启动续期，浏览器实机验证时再补写拦截层。

---

## 3. 快速开始（本机）

### 3.1 前置条件

| 依赖 | 本机状态 |
| --- | --- |
| Go 1.26 | ✅ |
| Node 22 | ✅ |
| MySQL 8.0 | ✅ **已作为 Windows 服务运行**（`D:\MySQL\MySQL Server 8.0`，3306 监听中） |
| Docker | ❌ **不可用** —— 本地开发不需要，M5 上线时才需要 |

本机 MySQL 为 **8.0.43**，`ngram_token_size=2` 已在 `my.ini` 配置（M3 中文全文检索直接可用，无需重启）。
数据库连接串已写入 `.env`（不入库）。

### 3.2 首次启动

```bash
# 1. 建库（在 MySQL 客户端执行）
CREATE DATABASE blog CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

# 2. 配环境（.env 已存在，含可用 DSN；换机器时改 DATABASE_URL 即可）
cp .env.example .env       # 若 .env 丢失才需要

# 3. 跑迁移
make migrate               # 25 个迁移

# 4. 灌示例文章
make seed                  # 导入 testdata/posts/*.md（幂等）

# 5. 起服务
make run                   # 后端 :8080

# 6. 另开终端起前端
make dev-web               # 前端 localhost:5173，已配 /api 代理到 :8080
```

> 没有数据库也能检查渲染效果：`make seed-dry`
>
> ⚠️ **访问地址用 `localhost:5173` 而不是 `127.0.0.1:5173`** —— Vite 8 在 Windows 上只监听 IPv6。
> 另外本机有系统代理，用 curl 测本地服务要加 `--noproxy '*'`。

### 3.3 常用命令

| 命令 | 作用 |
| --- | --- |
| `make run` | 启动后端（:8080） |
| `make migrate` / `migrate-down` / `migrate-status` | 迁移 up / down / 状态 |
| `make seed` / `make seed-dry` | 导入示例文章 / 只渲染不入库 |
| `make sqlc` | 由 `queries/*.sql` 重新生成 Go 代码 |
| `make css` | 由 chroma 重新生成代码高亮 CSS |
| `make dev-web` | 启动前端（:5173） |
| `make build` / `make web-build` | 构建后端 / 前端 |
| `make test` | 跑 Go 测试 |

---

## 4. 技术栈（已落地版本）

### 后端

| 组件 | 版本 / 选型 |
| --- | --- |
| Go | 1.26 |
| 路由 | `go-chi/chi/v5` v5.3.2 |
| 数据库 | MySQL 8.0（本机服务） |
| 驱动 | `go-sql-driver/mysql` |
| 迁移 | `pressly/goose` v3.28.0 |
| 配置 | `ilyakaznacheev/cleanenv` v1.5.0 |
| 代码生成 | `sqlc` v1.31.1（`engine: mysql`，`tinyint → int8`） |
| **Markdown** | `goldmark` v1.8.6 + GFM + Typographer |
| **代码高亮** | `chroma` v2.27.0（**class-based 输出**） |
| **XSS 消毒** | `bluemonday` v1.0.27 |
| **渲染缓存** | `hashicorp/golang-lru/v2` v2.0.7 |
| HTML 处理 | `golang.org/x/net/html` |
| ID 生成 | `oklog/ulid/v2` v2.1.2 |

### 前端

| 组件 | 版本 |
| --- | --- |
| React | 19.3 |
| Vite | 8.3 |
| TypeScript | 7.0 |
| Tailwind CSS | 4.3（`@tailwindcss/vite`） |
| react-router-dom | 7.18 |
| TanStack Query | 5.104 |
| Zustand | 5.0 |

**构建产物**：JS gzip **141.56 KB**（预算 150，125 模块）、CSS gzip 7.52 KB。

---

## 5. 目录结构（当前实际状态）

> 仓库根目录实际为 `E:/personal`（文档早期画成 `personal-blog/` 子目录，此处统一以项目名 `personal-blog/` 指代根）。

```
personal-blog/
├── cmd/
│   ├── server/main.go          # 服务入口（读配置 → 建依赖 → 起服务 → 优雅退出）
│   ├── migrate/main.go         # goose 迁移命令
│   ├── seed/main.go            # 示例文章导入（支持 -dry-run）
│   ├── gencss/main.go          # 生成 chroma 双主题 CSS
│   └── boot/main.go            # 启动自检/引导
├── internal/
│   ├── config/config.go        # 配置加载 + DSN 自动补齐（从 cwd 逐级向上找 .env）
│   ├── domain/                 # post.go / comment.go / media.go / admin.go + 泛型分页 Page[T]
│   ├── repository/
│   │   ├── pool.go             # 连接池（手写，不被 sqlc 覆盖）
│   │   ├── db.go / models.go / *.sql.go   # ⚠️ sqlc 生成，勿手写
│   │   ├── migrations/*.sql    # 25 个迁移（00001–00025）
│   │   └── queries/*.sql       # sqlc 源：posts / users / writing / admin / comments / media / refresh
│   ├── service/                # post.go / auth.go / comment.go / media.go / admin_post.go / actor.go / helpers.go
│   ├── transport/http/
│   │   ├── router.go
│   │   ├── post_handler.go / auth_handler.go / comment_handler.go
│   │   ├── media_handler.go / site_handler.go / admin_post_handler.go
│   │   └── middleware/{logging.go, auth.go}
│   └── pkg/
│       ├── markdown/           # 渲染管线：markdown.go / chroma.go / dom.go（含 _test.go）
│       ├── slug/               # slug.go（+ slug_test.go）
│       ├── media/media.go       # 本地磁盘存储（M2 起实装，非空）
│       ├── auth/auth.go         # JWT（HS256）/ argon2id 助手
│       └── apierr/apierr.go     # 统一错误体 {error:{code,message}}
├── testdata/posts/*.md         # 3 篇示例文章（带 front-matter）
├── web/src/
│   ├── main.tsx / vite-env.d.ts
│   ├── app/{theme.tsx, Providers.tsx}
│   ├── lib/{queryClient.ts, format.ts, api.ts}        # api.ts = coreFetch + 401 自动续期
│   ├── config/site.ts          # 博主资料兜底默认值（已被接口驱动取代，仅兜底）
│   ├── components/
│   │   ├── ThemeToggle.tsx / ProtectedRoute.tsx / AdminProtectedRoute.tsx
│   │   ├── Avatar.tsx / UserMenu.tsx / StudioUserMenu.tsx
│   │   └── TagCloud.tsx / ArchiveMini.tsx / SearchBox.tsx / ProfileCard.tsx
│   ├── pages/                  # LoginPage / RegisterPage / AboutPage / NotFoundPage / AdminLoginPage
│   ├── layouts/               # RootLayout / AdminLayout（纯管理）/ StudioLayout（我的博客）
│   ├── features/
│   │   ├── posts/             # api.ts / adminApi.ts / hooks.ts / hooks/useCodeCopy.ts
│   │   │                      # components/{PostCard,PostList,Pagination,Toc}.tsx
│   │   │                      # pages/{PostListPage,PostDetailPage,TagPage,ArchivePage,SearchPage,AdminPostListPage,PostEditorPage}.tsx
│   │   ├── media/             # api.ts / pages/MediaLibraryPage.tsx
│   │   ├── auth/              # store.ts / api.ts / types.ts / pages/{ProfilePage,UsersAdminPage}.tsx
│   │   ├── comments/          # api.ts / hooks.ts / components/CommentSection.tsx / pages/AdminCommentsPage.tsx
│   │   ├── users/             # api.ts / pages/UserBlogPage.tsx
│   │   └── site/              # useSiteProfile.ts（站长公开资料）
│   ├── styles/{index.css, chroma.css, prose.css}
│   └── router.tsx
├── docs/                       # 设计、开发、状态、部署与阶段记录
└── sqlc.yaml / Makefile / .env.example
```

> **⚠️ 不要碰**：`internal/repository/` 下的 `db.go`、`models.go`、`*.sql.go` 由 sqlc 生成并**静默覆盖**；`web/src/styles/chroma.css` 由 `make css` 生成。注意：`pkg/media` 已实装（非空）、`pkg/cache` 已删除。

---

## 6. 已建立的约定

| 约定 | 内容 |
| --- | --- |
| 数据库 | MySQL 8.0，**不要再提议换回 PostgreSQL** |
| 图片 | **本地磁盘**，不接对象存储，不预抽象 S3 接口 |
| 评论 | **无审核**，提交即公开；防垃圾前置到提交时 |
| 暗色模式 | **必须支持**，三态，首屏无闪烁 |
| 内容 | 全新写作，**不做迁移工具** |
| 部署 | 先本地落地；M5 **暂缓** |
| 代码高亮 | **chroma class-based**（原方案的 Shiki 是 TS 库，Go 侧用不了） |
| 时间 | 全链路 UTC |
| SQL 占位符 | `?`；无 `RETURNING`，用 `LastInsertId()` |
| 长文本 | `MEDIUMTEXT`；`toc_json` 用 `TEXT` |
| 迁移 | 一文件一 DDL + `NO TRANSACTION`，`down` 必须实测 |
| 已执行的迁移 | **不可改写**，一律追加新迁移（否则多环境 `goose_version` 对不上） |
| 审计字段 | `created_at` 由数据库 `DEFAULT CURRENT_TIMESTAMP(3)` 兜底；`updated_at` 带 `ON UPDATE`，**不靠调用点自觉传值** |
| 标签计数 | 不落库，由 `ListTagsWithCount` 的 JOIN 实时统计（`tags.post_count` 已删，见迁移 `00019`） |
| 渲染产物 | **h1 是主标题**，不进 TOC、不进摘要；正文从 h2 起 |
| 组件颜色 | **禁止硬编码**，只引用 CSS 变量 |
| 前端组织 | **feature-first**（`features/posts/{api,components,pages,hooks}`） |
| 权限分层 | **三级**：`is_admin`（纯管理后台 `/admin`）、`can_publish`（「我的博客」`/studio`）、`can_upload`（上传）；**普通注册用户默认 `can_publish=1, can_upload=0, is_admin=0`**（注册即作者，上传为管理员授予的特权，默认关闭） |
| 数据归属 | **行级隔离**：文章/媒体按 `author_id` 过滤与校验，判断统一走 `service.Actor`；`author_id=0` 的历史遗留数据仅管理员可管。跨用户写操作一律 403 |
| 个人资料 | 昵称/简介/头像/GitHub 存在 `users` 表，本人用 `PATCH /admin/me` 改（id 只取自 token）；站长公开资料走匿名 `GET /site/profile`，前端 `site.ts` 只是兜底 |

---

## 7. 文档体系

| 文档 | 回答的问题 |
| --- | --- |
| [`personal-blog-tech-plan.md`](./personal-blog-tech-plan.md) | **是什么、为什么这么选**（v1.11） |
| [`development-plan.md`](./development-plan.md) | **接下来做什么、验收标准是什么** |
| 本文件 `implementation-status.md` | **现在能跑什么、怎么跑、改过什么** |
| [`local-deploy.md`](./local-deploy.md) | **当前 Windows 环境如何部署、启动和验收** |
| [`docker-production-deploy.md`](./docker-production-deploy.md) | **Docker 线上服务器如何部署、升级、备份和排错** |
| [`devlog/Mx-*.md`](./devlog/) | **每阶段实际做了什么、踩了什么坑** |

**每阶段收尾的硬性 DoD**：更新 devlog → 更新本文件 → 勾选规划任务 → 偏离方案则改方案并升版本号 → 追加工作记忆。

---

## 8. 变更历史

| 日期 | 阶段 | 变更 |
| --- | --- | --- |
| 2026-09-28 | M1 | **前台界面补强**（用户反馈"缺少个人界面、页面太空旷"）：① 首页新增博主 Hero 区（头像/简介/社交/文章数）与右侧边栏（个人卡片含统计、标签云、最近写作，大屏 sticky）；② 新增 `/about` 关于页（个人介绍、本站技术栈、联系方式）；③ 导航加"首页/关于"项、品牌区改用博主名，页脚显示版权年份与备案号占位（`site.icp` 为空自动隐藏）；④ 文章卡片加 hover 反馈；⑤ 主容器加宽至 `max-w-6xl`，文章/标签/归档等窄页面各自限宽 `max-w-3xl`。**新增唯一配置点 `web/src/config/site.ts`**：博主名字/简介/头像/社交链接/建站年份/备案号全在这改，Hero、关于页、侧边栏、页脚共用。JS gzip 115.12 KB（预算 150 内）。个人身份信息暂存前端常量，不做后台编辑（改动频率以年计，值得做时再挪进数据库） |
| 2026-09-28 | M1 | **端到端验证闭环**：建库 + 22 个迁移全部 up + 灌 3 篇文章 + 6 个公开接口实测通过。同批修掉三个只有真跑起来才暴露的缺陷：① 审计字段缺默认值导致 INSERT 报错（迁移 `00014`–`00018`）；② h1 主标题混入摘要与目录；③ `tags.post_count` 是永远为 0 的冗余列（迁移 `00019` 删除）。详见 [devlog/M1-content-loop.md](./devlog/M1-content-loop.md) §9 |
| 2026-09-28 | M1 | 渲染管线（goldmark + chroma class-based + bluemonday + DOM 后处理 + LRU）；`pkg/apierr`、`pkg/slug`、`domain`、`service`、6 个公开接口；新增迁移 `00013`（`posts.toc_json`）；`cmd/seed` + 3 篇示例文章；`cmd/gencss` 生成双主题 CSS；前端 feature-first 重写（首页/详情页/标签/归档 + TOC 滚动高亮 + 代码复制）。详见 [devlog/M1-content-loop.md](./devlog/M1-content-loop.md) |
| 2026-09-29 | M2+M3 | **管理后台 + 评论 + 搜索全部落地并端到端验证** |
| 2026-09-29 | M2 补充 | **注册模块 + 权限分层** |
| 2026-09-29 | M2 补充（续） | 前端 `lib/api.ts` 补齐 **401 自动续期拦截**：任意请求 401 时用 refresh cookie 换新会话、并发去重后重试原请求；重试用独立超时控制器避免被原请求超时带崩。刷新失败清本地态回登录页。`api` 导出 `setRefreshHandler`，由 auth store 注册刷新回调。 |
| 2026-09-29 | M2 补充（续） | **编辑器保存反馈补强 + 前台跳转后台入口**：PostEditorPage 保存后显示「已于 hh:mm 保存」、同步表单发布状态、错误即时清空；`coreFetch` 重试独立超时；RootLayout 对 `can_publish` 用户显示「后台」入口，PostDetailPage 显示「编辑」链接直达 `/admin/posts/{public_id}`。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充小节 |
| 2026-09-29 | M2 补充（七）追记 | **工作台导航同样断链**（用户在 /studio/profile 实测后指出"个人资料界面还是有前往后台的按钮"）：`StudioLayout` 移除「后台」（→/admin）与「评论」（→/admin/comments）两个入口——评论管理在后台内部导航本就有，工作台不需要指路；导航只剩 文章/媒体库(按权限)/个人资料/前台。`AdminLayout` 的「我的博客/前台」返回链接保留（用户只限制"前往后台"方向）。grep 断言全前端无指向 /admin 的跳转链接；构建 121 模块（JS gzip 139.98 KB）。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充（七）追记 |
| 2026-09-29 | M2 补充（七） | **前台与后台彻底断链**（用户明确要求："移除前台界面的后台跳转功能，不管是管理员还是普通用户都不能直接跳转后台，后台需要单独访问"）：① 移除顶栏导航「我的博客」「后台」链接；② 用户菜单去掉「我的博客」「后台」条目（只留 我的主页/个人资料/退出）；③ 文章详情页移除「编辑」直达链接；④ 登录后不再按权限自动分流进 `/admin`/`/studio`，一律回来源页或首页。前台从此是纯内容站，工作台/后台只能直接输 URL 访问（守卫不变：未登录踢到登录页，登录后回来源页）。构建 121 模块（JS gzip 139.99 KB）。技术方案升 v1.10。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充（七） |
| 2026-09-30 | M2 补充（八） | **普通用户默认即作者，上传权限改由管理员授予**（用户要求"普通用户个人主页也应能编辑/发布文章，且上传权限由后台管理员判断"）：① 反转 v1.6 注册默认值——普通注册用户 `can_publish=1`（注册即作者）、`can_upload=0`（上传为管理员授予的特权，默认关闭）；② 现有普通用户（爱丽丝/鲍勃/joker/12/卡罗）置 `can_publish=1`；③ 本人查看自己的 `/blog/{publicID}` 主页时，资料头下方出现「写新文章 / 管理我的博客」入口（仅对自己可见，非后台入口）；④ 媒体库前端按 `can_upload` 显隐上传按钮，无权限显示提示；后端 `RequireUpload` 兜底；⑤ 用户管理页说明文案同步。前置基础设施（管理员「上传」开关、后端 `RequireUpload`）v1.6 已具备。curl 实测：新用户 `can_publish=true,can_upload=false` 可进 `/studio`；`can_upload=0` 令牌上传 403、`can_upload=1` 上传 201。构建 121 模块（JS gzip 140.53 KB）。技术方案升 v1.11。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充（八） |
| 2026-09-30 | M3 评论滚动分页 | **评论公开列表分页 + 折叠图标**（用户要求"文章下方加评论图标，点击查看历史评论，支持滑动分页加载"）：`GET /posts/{slug}/comments` 改为分页（返回带 `total` 的 `Page`），新增 `GET /posts/{slug}/comments/count` 仅返回公开评论总数；前端 `CommentSection` 默认折叠为「💬 评论（数量）」按钮，展开后用 `IntersectionObserver` 触底加载下一页（`useInfiniteQuery`）。`go build`/`npm run build`（122 模块）通过；curl 实测 count/分页正确。详见 [devlog/M3-comments-search.md](./devlog/M3-comments-search.md) |
| 2026-09-30 | M2 补充（九） | **首页去 Hero + 关于页标题视角 + 工作台去「前台」**（用户反馈"首页你好我是…可以去掉直接显示博客内容"、"关于页其他人视角应是关于作者"、"工作台移除前台按钮、上栏样式与前台一致"）：① `PostListPage` 移除顶部「你好，我是 XX」Hero，文章列表直接置顶；② `/about` 标题按视角：站长本人「关于我」、其他用户/匿名「关于作者」；③ `StudioLayout` 顶栏移除「前台」跳转按钮，上栏背景改为与前台一致的毛玻璃（`bg-bg/85 backdrop-blur`）。构建 122 模块（JS gzip 141.13 KB）。技术方案升 v1.12。 |
| 2026-09-30 | M2 补充（十） | **工作台导航：移除「媒体库」、增加「首页」**（用户反馈"移除资料库这个界面，上方增加首页按钮可以直接跳转首页"）：`StudioLayout` 顶栏导航改为 **首页 / 文章（按权限） / 个人资料**，移除「媒体库」入口（`/studio/media` 路由仍保留，用户可直接输 URL 访问）；「首页」作为返回前台的唯一出口。构建 122 模块（JS gzip 141.11 KB）。 |
| 2026-09-30 | M2 补充（十一） | **后台顶栏移除跳转按钮**（用户反馈"后台界面不应该有跳转前台的功能"）：`AdminLayout` 顶栏移除「我的博客」「前台」两个跳转链接，仅保留「用户 / 评论」；无权限提示页同步移除「返回博客首页」按钮，只留「退出登录」。后台与前台/工作台完全互不相链，三处界面均只认 URL 直访。构建 122 模块（JS gzip 141.11 KB）。 |
| 2026-09-30 | M2 补充（十二） | **用户只能编辑自己的博客 + 后台不再跳用户主页**（用户反馈"用户应该只能编辑自己的博客，后台还有直接跳转前台的按钮在用户管理那一块，管理员也不应该能直接登录或进入其他人的博客界面"）：① `service.Actor.AuthorFilter()` 改为始终返回当前用户 ID，站点管理员在 `/studio` 也只能看到/编辑自己的文章与媒体；跨用户写操作仍由 `CanTouch` 返回 403。② 用户管理页移除每行「博客」按钮，管理员不再能从后台直接跳转 `/blog/{public_id}`。`go build`/`npm run build`（122 模块，JS gzip 141.07 KB）通过。 |
| 2026-09-30 | M2 补充（十三） | **用户菜单「我的主页」改为返回网站首页**（用户反馈"用户点击我的主页和个人资料的跳转逻辑和界面布局有点问题"）：`UserMenu` 中「我的主页」由 `/blog/{public_id}` 改为 `/`；「个人资料」保持 `/studio/profile`。`npm run build`（122 模块，JS gzip 141.06 KB）通过。 |
| 2026-09-30 | M2 补充（十四） | **管理员独立登录页**（用户反馈"admin 应该有单独的登陆界面"）：新增 `/admin/login` 页面，作为管理员专属登录入口；未登录访问 `/admin/*` 时重定向到 `/admin/login`；登录成功后前端校验 `is_admin=true`，非管理员即使密码正确也会被拒绝并强制登出；普通用户继续使用 `/login`。`npm run build` 124 模块（JS gzip 141.34 KB）通过。 |
| 2026-09-30 | M2 补充（十五） | **工作台顶栏改为用户菜单下拉**（用户截图对比前台 UserMenu 与工作台顶栏，要求「以第一个图片为主」）：`StudioLayout` 顶栏导航精简为 **首页 / 文章（按权限）**；新增 `StudioUserMenu` 组件（与前台 `UserMenu` 样式一致），将「个人资料」「退出登录」收进右上角头像下拉菜单；保留「首页」作为返回前台的出口。`npm run build` 通过（125 模块，JS gzip 141.56 KB）。 |
| 2026-09-30 | M3 评论优化 | **评论昵称跟随用户且不可手动更改**（用户反馈"评论接口有点问题并且昵称应该跟随用户的昵称不支持手动更改"）：已登录用户提交评论时，`CommentSection` 自动使用 `display_name`（或邮箱前缀兜底）作为 `author_name`，隐藏昵称输入框，不可手动修改；匿名用户仍可自定义昵称（可选，默认「匿名」）。`npm run build` 124 模块（JS gzip 141.46 KB）通过。 |
| 2026-09-29 | M2 补充（六） | **普通用户的个人界面 + 注册校验收紧**（用户以新注册账号「12」实测后反馈"还是有问题"）：① `/studio/profile` 对所有登录用户开放——原 `StudioLayout` 把它也拦在 `can_publish` 墙后，普通用户点自己名字会被「您还没有发布权限」挡住、连资料都改不了（前端比后端严，拦错了地方）；② 顶栏身份区升级为**用户菜单** `UserMenu`（我的主页/个人资料/我的博客(仅 can_publish)/后台(仅 is_admin)/退出，点外部与 Esc 关闭）；③ 注册邮箱校验由「包含 @」收紧为**必须带 .后缀 的正则**（`12@123`/`a@b` → 400），前端 input 加 `pattern`。curl 实测：carol（普通用户）改资料 200、读文章接口 403、公开主页 200。构建 121 模块（JS gzip 140.15 KB）。技术方案 v1.9 补记。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充（六） |
| 2026-09-29 | M2 补充（五） | **用户模块与不同用户的公开界面**（用户反馈"还是有问题，用户模块与不同用户的相关界面"）：① 公开文章列表/详情带作者归属（`attachAuthors` 批量补全避免 N+1），卡片与详情页署名并可跳转；② 新增匿名接口 `GET /users/{publicID}` 与 `GET /users/{publicID}/posts`，前端新增 `/blog/:publicID` 用户博客主页（资料头 + 已发布文章）；③ 前台顶栏重做身份模块：匿名显示「登录/注册」（此前前台无登录入口），登录后显示当前用户头像昵称 + 「退出」；④ 用户管理页每行加「博客」直达该用户主页。curl 实测署名/404 全过；bob 发布演示文章一篇。技术方案升 v1.9。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充（五） |
| 2026-09-29 | M2 补充（四） | **数据按用户隔离 + 个人资料数据化**（用户反馈"不同用户登录看到的个人隐私应该不一样"）：① 文章/媒体按 `author_id` 行级隔离，跨用户改删 → 403（统一走新增的 `service.Actor`）；② 迁移 `00024` 加 `media.author_id`（0=历史遗留仅管理员可管）、`00025` 加 `users.bio/avatar_url/github`；③ 新增 `PATCH /admin/me`（本人改自己资料，id 只从 token 取）与匿名 `GET /site/profile`（站长公开资料，不含邮箱）；④ 评论管理由 `RequirePublish` 改为 `RequireAdmin`，入口从 `/studio/comments` 挪到 `/admin/comments`；⑤ 前端新增 `/studio/profile` 个人资料页、新增 `features/site/useSiteProfile.ts`，Hero/侧边栏/关于页/页脚与 StudioLayout 顶栏全部改由接口驱动，`config/site.ts` 退化为兜底默认值。curl 实测 9 项权限与隔离断言全过；`go build` + `npm run build`（118 模块）通过。技术方案升 v1.8。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充（四） |
| 2026-09-29 | M2 补充（三） | **后台与个人博客编辑界面彻底分离**：新增 `is_admin` 角色（迁移 00023），权限分三级——`is_admin`（纯管理后台 `/admin`）、`can_publish`（个人博客工作台 `/studio`）、`can_upload`（上传）。用户管理接口与页面改由 `RequireAdmin` 保护，新增「管理员」开关可升降某人后台权限；文章/媒体/评论留在 `/studio` 工作台。前端新增 `StudioLayout`、重写 `AdminLayout` 为纯管理、`RootLayout` 导航按权限分别显示「我的博客」与「后台」。技术方案升 v1.7。curl 实测：管理员两处皆 200、非管理员作者仅 `/studio` 接口 200、`/admin` 接口 403。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充小节 |
| 2026-09-29 | M2 补充 | **注册模块 + 权限分层**：公开注册接口（注册即登录）、`users.can_publish` 列（迁移 00022）、`RequirePublish` 中间件、用户管理接口与页面；普通注册用户默认可上传不可发帖，发帖权限由管理员开启。技术方案升 v1.6。详见 [devlog/M2-admin-backend.md](./devlog/M2-admin-backend.md) 补充小节 |
| 2026-09-28 | M0 | 初始化工程：Go 骨架、12 个迁移、sqlc 打通；前端骨架（含无闪烁暗色模式）；Git 初始化；建立项目文档体系。详见 [devlog/M0-foundation.md](./devlog/M0-foundation.md) |

---

## 9. 待办与阻塞项

| 项 | 类型 | 说明 |
| --- | --- | --- |
| 迁移 `down` 未逐个实测 | 🟡 | 回滚路径未验证；下次动 DDL 前先 `make migrate-down` 演练 |
| bimg/libvips 在 Windows 的编译 | ✅ 已解决（M2） | 确认装不上，降级为 `golang.org/x/image` 原生处理（保留原格式 + GIF 直出 + 超宽缩放），已记技术方案 v1.5 |
| 前端未做浏览器实机视觉验证 | 🟡 | 起服务后人工过一遍样式细节 |
| ~~MySQL 密码（阻塞 M1 验收）~~ | ✅ 已解决 | 库已建、迁移已跑、端到端已验证 |
| ~~`ngram_token_size` 未确认~~ | ✅ 已解决 | 本机 `my.ini` 已是 2，无需改动 |
| M5 部署文档已产出但未实际上线 | 🟡 | [`docker-production-deploy.md`](./docker-production-deploy.md) 与根目录 Dockerfile / `docker-compose.prod.yml` / Caddyfile 已落地；尚未在真实服务器演练；采用 Caddy 直出 `web/dist`，无需改 Go 代码。旧版 [`deploy.md`](./deploy.md) 保留作方案记录 |
