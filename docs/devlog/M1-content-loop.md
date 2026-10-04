# M1 · 文章核心闭环 — 落地记录

> 状态：**✅ 已完成并端到端验证** ｜ 日期：2026-09-28 ｜ 分支：`main`
>
> 对应规划：[development-plan.md](../development-plan.md) §4
>
> **补记（同日）**：拿到数据库连接后完成端到端验证，并修掉了三个只有真跑起来才会暴露的缺陷
> （审计字段缺默认值、h1 混入摘要与目录、`tags.post_count` 冗余列）。详见 §5 坑 7–9 与 §9。

---

## 1. 完成情况

| 分类 | 任务 | 状态 |
| --- | --- | --- |
| 后端 | M1-01 ~ M1-07 渲染管线（goldmark / chroma / bluemonday / anchor / 外链 / 图片 / LRU） | ✅ |
| 后端 | M1-08 ~ M1-09 查询补全 + sqlc 生成 | ✅ |
| 后端 | M1-10 service 层 | ✅ |
| 后端 | M1-11 HTTP handlers | ✅ |
| 后端 | M1-12 `pkg/apierr` 统一响应 | ✅ |
| 后端 | M1-13 `cmd/seed` 示例数据导入 | ✅ |
| 前端 | M1-14 ~ M1-21 API 客户端 / 首页 / 详情页 / 标签页 / 归档页 / 主题 / 双主题 CSS / 各状态 | ✅ |

**21/21 全部实现，端到端已跑通**（库建好、19 个迁移全部 up、3 篇示例文章入库、6 个公开接口实测返回正确数据）。

---

## 2. 三个待决策项的结论

| 编号 | 结论 | 理由 |
| --- | --- | --- |
| **M1-D1** 代码高亮 | **Go 侧用 chroma，输出 class-based HTML**；亮/暗两套 CSS 由 `cmd/gencss` 生成，暗色挂在 `.dark .chroma` 下 | 原方案写的 Shiki 是 TypeScript 库，Go 侧调用不了。chroma 的 class-based 输出让「切主题」退化成「换 CSS 变量」，**零 JS、零重新渲染**，比内联样式的方案更干净 |
| **M1-D2** 中文 slug | **保留中文**。`slug.Make("Go 并发模式：从入门到放弃")` → `go-并发模式-从入门到放弃` | 转拼音后可读性反而更差；现代浏览器会把 UTF-8 URL 显示成中文，搜索引擎也能正确处理。需要短链时作者可在编辑器里手动覆盖 |
| **M1-D3** 示例数据 | **`cmd/seed` + `testdata/posts/*.md`**（3 篇真实技术文章，带 front-matter） | M2 才有编辑器；这批 Markdown 同时充当渲染管线的测试样本。seed 按 slug 判重，**幂等**，可反复跑 |

---

## 3. 实际产出

### 3.1 后端

| 文件 | 内容 |
| --- | --- |
| `internal/pkg/markdown/markdown.go` | 渲染主体：goldmark + GFM + Typographer → bluemonday 消毒 → DOM 后处理；LRU 缓存（key = sha256 原文，容量 256） |
| `internal/pkg/markdown/chroma.go` | chroma class-based 输出器 + `.code-block` 容器 + 兜底转义 |
| `internal/pkg/markdown/dom.go` | 后处理：heading anchor（保留中文、重复自动去重）、外链 `target/rel`、图片 `loading/width/height`、单图段落升级为 `<figure>`；同时产出 TOC 与纯文本 |
| `internal/pkg/slug/slug.go` | slug 生成、长度上限 96、占用时追加序号 |
| `internal/pkg/apierr/apierr.go` | 统一 `{data}` / `{error:{code,message}}`；8 个错误码；5xx 只回通用文案，内部原因只进日志 |
| `internal/domain/post.go` | 实体定义，含泛型分页容器 `Page[T]` |
| `internal/service/post.go` | 列表 / 详情 / 标签 / 归档 / 按月；批量补标签（避免 N+1）；分页参数越界拉回合法区间而非报错 |
| `internal/transport/http/post_handler.go` | 6 个公开接口 |
| `cmd/gencss/main.go` | 生成 `web/src/styles/chroma.css`（`make css`） |
| `cmd/seed/main.go` | 导入示例文章；支持 `-dry-run`（不连库，只渲染并打印） |
| `queries/posts.sql` | 12 条读查询 |
| `queries/writing.sql` | 7 条写查询（seed 用，M2 后台复用） |
| `migrations/00013_add_posts_toc.sql` | **新增 `posts.toc_json`** |

**公开接口**：

```
GET /api/v1/posts?page=1&page_size=10
GET /api/v1/posts/{slug}
GET /api/v1/tags
GET /api/v1/tags/{slug}/posts
GET /api/v1/archive
GET /api/v1/archive/{year}/{month}
```

### 3.2 前端

feature-first 组织，全部在 `web/src/features/posts/`：

```
api.ts            类型 + fetch 封装（统一超时/错误码/解包）
hooks.ts          TanStack Query hooks + 统一 queryKey
hooks/useCodeCopy.ts    给 .code-block 注入复制按钮
components/       PostCard / PostList / Pagination / Toc（含滚动高亮）
pages/            PostListPage / PostDetailPage / TagPage / ArchivePage
```

样式：`styles/chroma.css`（生成）+ `styles/prose.css`（正文排版，手写，不引 typography 插件）+ `index.css` 里用 `@theme inline` 把 CSS 变量桥接成 Tailwind 工具类。

---

## 4. 关键决策

| 决策 | 理由 |
| --- | --- |
| **TOC 落库，不在详情页实时解析** | 实时解析等于每次请求都跑一遍 DOM 解析；TOC 与正文同属一次渲染产物，一起写库才能保证永不脱节。为此新增迁移 `00013`（`toc_json TEXT`） |
| **消毒放在后处理之前** | 后处理只添加我们自己构造的属性，所有来自 Markdown 的内容都要先过白名单。顺序反了等于给 XSS 留口子 |
| **列表页批量查标签，不用 `GROUP_CONCAT`** | 两个原因：sqlc 会把它推断成 `interface{}`；更关键的是 `GROUP_CONCAT` 有 `group_concat_max_len`（默认 1024 字节）截断风险。改成两步查询 + `sqlc.slice` |
| **PlainText 跳过块级代码，保留行内代码** | 全跳过会让摘要断成「最朴素的写法是用 加一个 变量」；全保留会让摘要混进大段代码。块级代码依然能被搜到（搜索走 `content_md` 原文） |
| **分页参数越界拉回默认值，不报 400** | 分页不是业务输入，访客改坏了拉回默认值体验更好 |
| **`cmd/seed` 用占位作者且 password_hash 故意非法** | `!placeholder-not-loginable` 不以 argon2 开头，M2 的密码校验会直接拒绝 —— 不存在「默认密码」风险 |
| **不引入 `@tailwindcss/typography`** | 需要的规则不到一百行，手写还能直接吃项目 CSS 变量，与主题切换天然一致 |

---

## 5. 踩坑记录

### 坑 1：chroma 没有 `github-light` 样式 ⭐ 最隐蔽

照直觉写 `styles.Get("github-light")`，**不报错**，静默返回黑底的 Fallback 样式。结果「亮色主题」生成出来是黑底，只有看生成的 CSS 才发现。

**实际可用名**：`github`（亮）、`github-dark`（暗）。已写进 `cmd/gencss` 的注释。

### 坑 2：sqlc 不生成 `Post` 模型

因为 `omit_unused_structs: true` + 查询里显式列出列名，sqlc 只生成 `GetPostBySlugRow` 这类结构，`repository.Post` 根本不存在。service 里按模型名写会编译不过。

**对策**：service 引用具体的 `XxxRow` 类型。另外 sqlc 为**每个查询**生成独立的 Row 类型，即使列完全相同也不能互通 —— 与其做泛型体操，不如写三个转换函数（各 6 行）。

### 坑 3：`GROUP_CONCAT` 子查询被推断成 `interface{}`

见 §4 第三条。

### 坑 4：goldmark 覆盖渲染器的优先级方向

`NodeRenderers.Sort()` 是**升序**排序，而注册时**倒序遍历** —— 所以**优先级低**的渲染器后注册、反而覆盖内置的（内置是 1000）。自定义代码块渲染器要用 `util.Prioritized(x, 100)`。这个方向不查源码一定会搞反。

### 坑 5：slug 的连续连字符没合并

`a---b` 原样输出 `a---b`。修法：连字符也走 pending 逻辑，连续分隔符只保留一个。

### 坑 6：`range over unicode.ToLower(r)`

`ToLower` 返回的是 `rune` 不是 `string`，不能用 `for range`。直接 `WriteRune(unicode.ToLower(r))`。

### 坑 7：审计字段没有默认值，INSERT 直接报错 ⭐ 只有真连库才会暴露

`seed` 第一次跑就挂了：`Error 1364: Field 'created_at' doesn't have a default value`。

所有表的 `created_at` / `updated_at` 都定义为 `DATETIME(3) NOT NULL` 但**没有 DEFAULT**。
MySQL 严格模式下，任何一条 INSERT 少传一个就报错。`CreatePost` 侥幸没挂只是因为它恰好传了，
`CreateUser` 没传就炸了 —— 说明这不是「某条 SQL 写漏了」，而是 **schema 层没有兜底**。

**修法**：新增迁移 `00014`–`00018`，给 5 张表的审计字段加
`DEFAULT CURRENT_TIMESTAMP(3)`，并给 `updated_at` 加 `ON UPDATE CURRENT_TIMESTAMP(3)`
（省掉每条 UPDATE 都要记得写 `updated_at = ?`，漏写是这类字段最常见的 bug）。

**为什么不改 `00001` 而是要追加 ALTER**：`00001` 已经执行过了。改写已执行的迁移是反模式 ——
一旦存在第二个环境，两边的 `goose_version` 就对不上。宁可多 5 个迁移文件。

### 坑 8：h1（主标题）混进了摘要和目录 ⭐ 端到端才看得见

列表接口返回的摘要长这样：

```
MySQL 中文全文检索：ngram 的两个坑 MySQL 8.0 内置了 ngram 分词器……
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ 标题被重复了一遍
```

同时侧边目录的第一项也是文章标题。

根因同一个：`postProcess` 把 h1 当正文处理了 —— 它的文本进了 PlainText，也被收进了 TOC。

**修法**：`shouldSkipText` 增加 `h1`；TOC 只收 `level >= 2`。
h1 的 anchor id 仍然保留（文内可能有指向它的链接）。

已补两个单测锁住行为，并特意把「行内代码保留 / 块级代码跳过」这对**方向相反**的规则放在一起测，
因为它们极易互相改坏。

### 坑 9：`tags.post_count` 是个永远为 0 的冗余列

标签计数实际由 `ListTagsWithCount` 用 `JOIN + COUNT` 实时算出（只统计已发布文章），
而 `tags.post_count` 这个物理列**从未被任何代码写入**，永远是 0。

它的危险在于**看起来是对的**：列在、类型对、名字合理，将来谁写一句
`SELECT post_count FROM tags`（不 JOIN）会拿到 0 而不起疑。

**修法**：迁移 `00019` 删列。作为缓存它在本项目规模下没有意义（<1000 篇文章），
维护成本却很高（发布/转草稿/删除都要重算）。

### 坑 10：Vite 8 在 Windows 只监听 IPv6

`vite dev` 起来后 `curl 127.0.0.1:5173` 连不上，但 `localhost` / `[::1]:5173` 正常 ——
Vite 8 默认只绑定 `[::1]`。另外本机有系统代理（`http_proxy=127.0.0.1:49334`），
curl 会走代理，测本地服务必须加 `--noproxy '*'`，否则拿到的是代理返回的 502，很误导。

---

## 6. 验证结果

| 项 | 命令 | 结果 |
| --- | --- | --- |
| 后端编译 | `go build ./...` | ✅ |
| 静态检查 | `go vet ./...` | ✅ |
| 渲染管线单测 | `go test ./internal/pkg/...` | ✅ 10 个用例全过（高亮 class-based / 中文 anchor / 外链 / 图片 / XSS 拦截 / GFM 表格与任务列表 / 缓存 / 摘要 / **h1 不进摘要与目录** / **行内代码保留·块级代码跳过**） |
| slug 单测 | `go test ./internal/pkg/...` | ✅ 3 个用例 |
| 前端类型检查 + 构建 | `npm run build` | ✅ JS **gzip 112.64 KB**（预算 150）、CSS gzip 4.46 KB |
| 渲染效果（真实文章） | `make seed-dry` | ✅ 3 篇文章：目录层级正确、中文 anchor 正常、摘要干净、阅读时长合理 |

### 端到端验证（已完成）

环境：本机 MySQL 8.0.43（Windows 服务，`D:\MySQL\MySQL Server 8.0`），`ngram_token_size=2` **已就绪**，无需重启。

| 步骤 | 结果 |
| --- | --- |
| 建库 `blog`（utf8mb4 / `utf8mb4_0900_ai_ci`） | ✅ |
| `make migrate`（19 个迁移全部 up） | ✅ 含 ngram FULLTEXT 索引 |
| `cmd/seed` 灌 3 篇示例文章 | ✅ 成功 3 / 失败 0 |
| **seed 幂等性**（连跑两次） | ✅ posts 仍为 3，未重复插入；标签关联未重复累加 |
| `/healthz` `/readyz` | ✅ 200（`readyz` 查库通过，证明连通） |
| `GET /api/v1/posts` | ✅ 列表 + 分页 + 标签 |
| `GET /api/v1/posts/{slug}` | ✅ 正文 HTML + TOC（仅 h2 起）+ 摘要（不含标题） |
| `GET /api/v1/tags` | ✅ 6 个标签，计数正确（JOIN 实时统计） |
| `GET /api/v1/tags/{slug}/posts` | ✅ |
| `GET /api/v1/archive`、`/archive/2026/9` | ✅ 按月归档正确 |
| **404 错误路径** | ✅ `{"error":{"code":"not_found",...}}` + HTTP 404 |
| **分页越界** | ✅ 返回空 items，total 仍为 3，不报错 |
| **前端 → 后端代理** | ✅ `localhost:5173/api/v1/*` 正常透传到 `:8080` |
| 前后端字段契约 | ✅ 逐字段核对一致（`public_id` / `content_html` / `toc` / `comment_enabled` 等） |

`make seed-dry` 的实际输出片段：

```
=== 02-go-concurrency-errgroup.md ===
title       : Go 并发：errgroup 与有界并发
slug        : go-concurrency-errgroup
tags        : [Go 并发]
正文 1899 字符 / HTML 17718 字符 / 目录 5 项 / 阅读约 4 分钟
  - Go 并发：errgroup 与有界并发 (#go-并发-errgroup-与有界并发)
    - 场景一：一组任务，任一失败就整体取消 (#场景一-一组任务-任一失败就整体取消)
    ...
```

---

## 7. 遗留问题

| 项 | 影响 | 处理 |
| --- | --- | --- |
| 迁移 `down` 未逐个实测 | 回滚路径未验证 | 当前库无关键数据；下次动 DDL 前先 `make migrate-down` 演练 |
| bimg/libvips 在 Windows 的编译 | M2 图片上传可能受阻 | 装不上则降级 Go 原生 image 处理 |
| 前端未做浏览器实机视觉验证 | 样式细节可能有偏差 | 起服务后人工过一遍（本次只验证了接口契约与构建） |
| ~~MySQL 密码未提供~~ | ~~阻塞端到端验证~~ | ✅ 已解决 |

---

## 8. 对技术方案的回写

- §7.1 渲染管线：高亮由 Shiki 改为 **chroma（class-based）**，并说明为什么要改（Shiki 是 TS 库，Go 侧用不了）
- §4 数据模型：新增 `posts.toc_json`；`00019` 删除 `tags.post_count`；`00014`–`00018` 补审计字段默认值
- §6.5 暗色模式：补充「代码高亮双主题的实现方式」
- 版本号 v1.3 → **v1.4**

---

## 9. 本轮补记：环境事实

后续会话直接引用，不必重新探测：

- 本机 MySQL **8.0.43** 以 Windows 服务运行，路径 `D:\MySQL\MySQL Server 8.0`
- **连接串已写入 `.env`**（该文件不入库）：`root@tcp(127.0.0.1:3306)/blog`
- `ngram_token_size=2` 已在 `my.ini` 配好 —— 这是只读启动参数，M3 做中文全文检索时不用动它
- **本机没有 Docker**，本地开发一律直连本机 MySQL；docker-compose 那套留到 M5 上线再用
- 一键起本地环境：`make run`（后端 `:8080`）+ `make dev-web`（前端 `localhost:5173`，**注意只监听 IPv6**）
- 本机有系统代理（`http_proxy=127.0.0.1:49334`），用 curl 测本地服务**必须加 `--noproxy '*'`**，否则拿到代理的 502 会误判成服务没起来
