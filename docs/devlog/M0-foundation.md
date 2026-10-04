# M0 · 工程地基 — 落地记录

> 状态：✅ 完成 ｜ 日期：2026-09-28 ｜ 分支：`main`
>
> 对应规划：[development-plan.md](../development-plan.md) §1

---

## 1. 目标与完成情况

| 规划任务 | 状态 | 说明 |
| --- | --- | --- |
| 初始化 Git 仓库、Makefile | ✅ | |
| `sqlc.yaml` 配 `engine: mysql`，跑通"写 SQL → 生成 Go" | ✅ | sqlc v1.31.1 |
| Go 工程骨架：配置、健康检查、slog、优雅退出 | ✅ | |
| 前端骨架：Vite + TS + Tailwind + 路由 + QueryClient | ✅ | |
| CI：push 跑 test/vet/build | ⬜ **跳过** | 本地阶段暂无远端仓库，留到 M5 |

**5/5 完成（CI 一项主动跳过，理由见下）**。

---

## 2. 环境探测结论（重要）

开工前摸了本机环境，有两个结论直接改变了做法：

| 发现 | 影响 |
| --- | --- |
| **MySQL 8.0 已作为 Windows 服务运行**（`D:\MySQL\MySQL Server 8.0`，PID 3548 监听 3306） | 本地开发**直接连本机服务**，不需要起容器 |
| **Docker / docker compose 不可用** | 方案附录 D 的 `docker-compose.yml` 与 `make dev-db` 在本机**不适用**；M0–M4 不受影响，M5 上线时再启用 |

> 因此附录 A 里的 `make dev-db`（起 MySQL 容器）已从本地流程中移除，Makefile 里也明确写了"本机 MySQL 已作为服务运行，不需要 docker"。

---

## 3. 实际产出

### 3.1 后端

| 文件 | 内容 |
| --- | --- |
| `cmd/server/main.go` | 装配顺序：配置 → slog → 连接池 → HTTP server → 优雅退出（SIGINT/SIGTERM，等在途请求 ≤10s） |
| `cmd/migrate/main.go` | goose 的 up / down / status / reset |
| `internal/config/config.go` | cleanenv 加载 + **DSN 参数自动补齐** |
| `internal/repository/pool.go` | 连接池配置（max open/idle/lifetime） |
| `internal/transport/http/router.go` | chi + RequestID + Recoverer + 结构化日志 |
| `internal/transport/http/middleware/logging.go` | slog 请求日志 |
| `internal/repository/migrations/*.sql` | **12 个迁移文件** |
| `internal/repository/queries/{posts,users}.sql` | sqlc 查询源 |

**迁移文件清单**（一文件一 DDL + `NO TRANSACTION`）：

| # | 内容 |
| --- | --- |
| 00001 | `users` |
| 00002 | `posts` |
| 00003 | `tags` |
| 00004 | `post_tags` |
| 00005 | `comments` |
| 00006 | `media` |
| 00007 | `post_views` |
| 00008 | `refresh_tokens` |
| 00009 | 索引 `posts(status, published_at DESC)` |
| 00010 | 索引 `post_tags(tag_id)` |
| 00011 | 索引 `comments(post_id)` |
| 00012 | **FULLTEXT `WITH PARSER ngram`** |

**暴露的端点**：

- `GET /healthz` —— 存活探针，不查依赖
- `GET /readyz` —— 就绪探针，查库连通性（2s 超时）
- `GET /api/v1/ping`

### 3.2 前端

Vite 8.3 + React 19.3 + TS 7.0 + Tailwind v4 + react-router 7.18 + TanStack Query 5.104 + Zustand 5.0。

- 路由骨架 + RootLayout + 首页/404 占位页
- 主题系统（对应方案 §6.5）：CSS 变量色板 + 三态切换 + localStorage
- **首屏无闪烁脚本写在 `index.html` 的 `<head>` 里**（放 `useEffect` 必然闪白）

### 3.3 工程配置

`sqlc.yaml`（`engine: mysql`，`tinyint → int8` 覆盖）、`Makefile`、` .env.example`、`.gitignore`、Git 初始化。

---

## 4. 关键决策

| 决策 | 理由 |
| --- | --- |
| **迁移用独立命令 `cmd/migrate`，不放在服务启动时执行** | 避免多实例竞态；失败时一眼可见，不会混入服务日志 |
| **DSN 的三个参数由代码自动补齐，不依赖人填** | `parseTime=true` / `loc=UTC` / `charset=utf8mb4` 少一个就会埋雷（时间扫描报错、emoji 存不进）。与其写进文档让人记，不如代码兜住 |
| **连接池代码放 `pool.go`，不叫 `db.go`** | 见下方踩坑 1 |
| **CI 本阶段跳过** | 本地阶段没有远端仓库，配了也无法触发；留到 M5 一并做 |

---

## 5. 踩坑记录

### 坑 1：sqlc 会静默覆盖手写文件 ⭐ 最重要

`sqlc generate` 会往 `out` 目录写 **`db.go`、`models.go`、`*.sql.go`**。手写的 `internal/repository/db.go`（含 `Open()` 连接池逻辑）被**整个替换**，编译期才发现 `repository.Open` 消失。

**对策**：持久化层里手写的代码**避开这三个文件名**，连接池改放 `pool.go`。
**已同步**：技术方案 §4 加了一行"⚠️ sqlc 的覆盖陷阱（M0 实操已踩）"。

### 坑 2：新版 TypeScript 移除了 `baseUrl`

`tsconfig.json` 里的 `"baseUrl": "."` 报 unknown option。改用相对路径 `./src/*` 配置 `paths`。

### 坑 3：Vite 工程缺 Node 类型

`vite.config.ts` 用到了 Node API 却没有 `@types/node`，且缺 `src/vite-env.d.ts`。补装 `@types/node` 并加上 `/// <reference types="vite/client" />`。

### 坑 4：main.go 的 import 在 Windows 下需清理

`syscall` 与未使用的 `path/filepath` 导致编译失败。改用 `os/signal` 的标准做法处理退出信号。

---

## 6. 验证结果

| 项 | 命令 | 结果 |
| --- | --- | --- |
| 后端编译 | `go build ./...` | ✅ |
| 静态检查 | `go vet ./...` | ✅ |
| sqlc 生成 | `sqlc generate` | ✅ 生成 models / posts / users，`tinyint → int8` 生效 |
| 前端类型检查 + 构建 | `npm run build` | ✅ 首屏 JS **gzip 109.66 KB**（预算 150 KB） |
| **配置链路 + DB 可达性** | 用错误密码起服务 | ✅ 输出 `Access denied for user 'root'@'localhost'` —— 说明 MySQL 可达、配置加载与 DSN 归一化均正常，**只差正确密码** |

> 迁移**尚未实际执行过**（未拿到 MySQL 密码）。这是 M1 的第一个阻塞项。

---

## 7. 遗留问题

| 项 | 影响 | 处理 |
| --- | --- | --- |
| **MySQL root 密码未提供** | 阻塞 M1：迁移跑不了、seed 灌不了数据 | 需要用户提供，或用户自行 `CREATE DATABASE blog` 后给出 DSN |
| CI 未配置 | 无自动化校验 | M5 一并做 |
| 迁移文件未实测 `down` | 回滚路径未验证 | 拿到密码后逐个实测 |
| `ngram_token_size` 未确认 | M3 阶段 2 可能受阻 | 检查本机 `my.ini` |

---

## 8. 对技术方案的回写

本阶段对 `docs/personal-blog-tech-plan.md` 的改动：

- §4 增加"⚠️ sqlc 的覆盖陷阱（M0 实操已踩）"
- §6 前端目录策略改为 **feature-first**
- 附录 A 命令表按"本机无 Docker"重写
- 版本号 v1.1 → **v1.2**
