# 本地项目部署说明

> 适用项目：个人博客（Go API + React/Vite + MySQL 8.0）
> 适用环境：Windows 10/11、PowerShell、本仓库根目录 D:\personal
> 文档版本：2026-10-04

## 1. 文档边界

仓库中的 personal-blog-tech-plan.md 是技术设计和选型说明，不是逐条可执行的启动脚本；docs/deploy.md 面向 Linux 云主机的 Docker Compose + Caddy 生产部署。本文件只说明当前 Windows 环境下的本地开发和联调流程。

以代码和 docs/implementation-status.md 为准，当前本地拓扑如下：

~~~text
浏览器 http://localhost:5173
        │
        │ Vite 代理 /api、/uploads
        ▼
Go API http://127.0.0.1:8080
        │
        ▼
MySQL 8.0 127.0.0.1:3306 / blog
~~~

本地不需要 Docker。Docker Compose 只用于后续线上部署。

## 2. 当前环境检查结果

| 组件 | 项目要求 | 当前检查结果 | 结论 |
| --- | --- | --- | --- |
| Go | 1.26.0 或更高 | go.exe 为 1.24.5 | **必须先升级**，否则 Go 命令会被 go.mod 拒绝 |
| Node.js | 支持 Vite 8，建议 Node 22 | 22.17.1 | 可用 |
| npm | 与 Node 配套 | 10.9.2 | 可用；PowerShell 下使用 npm.cmd |
| MySQL | 8.0 | Windows 服务 MySQL80 正在运行 | 服务可用性正常，账号密码需单独验证 |
| Docker | 线上才需要 | Docker daemon 当前不可用 | 本地流程不依赖 |

当前已经验证：web 前端执行 npm.cmd run build 成功（125 个模块）。当前未能执行 Go 测试/迁移，因为本机 Go 版本低于 go.mod 的 1.26.0 要求；MySQL 服务虽在运行，但现有 .env 中的连接凭据返回了 ERROR 1045 Access denied，需要先修正密码或改用有权限的本地账号。

## 3. 前置条件

### 3.1 安装或切换 Go

安装 Go 1.26.0 或更高版本，并确认 PowerShell 找到的是新版本：

~~~powershell
go version
~~~

预期输出类似 go version go1.26.x windows/amd64。如果仍显示 1.24.5，检查 PATH 中 D:\GO\bin 是否排在旧 Go 路径之后，或重启终端。

本项目不要为了绕过报错而直接降低 go.mod 的 Go 版本；项目依赖和代码以 1.26.0 为基线。Go 的自动工具链下载在当前机器上还受到 C:\Users\pc\go 权限限制，优先安装本机 Go 1.26，而不是依赖自动下载。

### 3.2 Node.js 和前端依赖

确认版本：

~~~powershell
node --version
npm.cmd --version
~~~

PowerShell 可能禁止执行 npm.ps1，所以本文统一使用 npm.cmd，不需要修改执行策略。

### 3.3 MySQL 账号和数据库

确认服务：

~~~powershell
Get-Service MySQL80
~~~

服务状态应为 Running。使用 MySQL 客户端登录（会交互式询问密码，不把密码写进命令历史）：

~~~powershell
& 'D:\MySQLFiles\MySQL Server 8.0\bin\mysql.exe' -h 127.0.0.1 -P 3306 -u root -p
~~~

登录后创建数据库：

~~~sql
CREATE DATABASE IF NOT EXISTS blog
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
~~~

如果 root 登录仍返回 ERROR 1045，先在 MySQL 管理工具中重置密码，或创建一个仅用于本地博客的账号，再把 .env 的 DATABASE_URL 改成该账号。不要把真实密码提交到 Git。

## 4. 配置 .env

仓库根目录已有 .env 时直接检查它；换机器或文件缺失时执行：

~~~powershell
Copy-Item .env.example .env
~~~

至少确认以下值与本机一致：

~~~dotenv
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=blog_user:密码@tcp(127.0.0.1:3306)/blog
JWT_SECRET=至少32字节的固定随机字符串
MEDIA_DIR=./data/uploads
MEDIA_URL_PREFIX=/uploads
BASE_URL=http://localhost:8080
CORS_ORIGINS=http://localhost:5173
~~~

说明：

- 程序会自动补齐 MySQL DSN 的 parseTime=true、loc=UTC 和 charset=utf8mb4。
- JWT_SECRET 必须固定；服务重启后如果变更，已有登录令牌会全部失效。
- MEDIA_DIR 会被锚定到 .env 所在的仓库根目录，上传文件落在 data/uploads。
- .env 已在 .gitignore 中，禁止提交。

## 5. 首次部署流程

以下命令均在仓库根目录 D:\personal 执行。先完成第 3、4 节的 Go 和数据库修正。

### 5.1 安装前端依赖

~~~powershell
Push-Location web
& npm.cmd ci
Pop-Location
~~~

### 5.2 执行数据库迁移

~~~powershell
go run ./cmd/migrate -cmd status
go run ./cmd/migrate -cmd up
~~~

当前仓库包含 25 个追加式迁移，首次 up 会创建用户、文章、标签、媒体、评论、刷新令牌及全文索引等表。迁移文件的 DDL 在 MySQL 下会隐式提交，开发时不要改写已经执行过的迁移文件。

### 5.3 导入示例文章（可选）

~~~powershell
go run ./cmd/seed
~~~

该命令读取 testdata/posts/*.md，按 slug 幂等写入；可以重复执行。只想验证 Markdown 渲染而不连接数据库时：

~~~powershell
go run ./cmd/seed -dry-run
~~~

### 5.4 创建首个管理员

首次创建管理员时不要使用代码中的默认密码，使用当前 PowerShell 会话变量：

~~~powershell
$env:ADMIN_EMAIL = 'admin@example.com'
$env:ADMIN_PASSWORD = '请替换为至少8位的强密码'
go run ./cmd/boot
Remove-Item Env:ADMIN_EMAIL, Env:ADMIN_PASSWORD
~~~

也可以显式传参：

~~~powershell
go run ./cmd/boot -email admin@example.com -password '请替换为强密码' -name '站长'
~~~

管理员后台地址是 http://localhost:5173/admin/login；普通用户登录地址是 http://localhost:5173/login。前台不会显示后台跳转入口，/admin 和 /studio 需要直接输入 URL。

## 6. 启动本地服务

### 6.1 启动 Go API

终端 A，在仓库根目录执行：

~~~powershell
go run ./cmd/server
~~~

默认监听 :8080。服务启动时会自动创建 data/uploads。

### 6.2 启动 Vite 前端

终端 B 执行：

~~~powershell
Push-Location web
& npm.cmd run dev
~~~

浏览器打开 http://localhost:5173。Vite 配置会把 /api/* 和 /uploads/* 代理到 http://127.0.0.1:8080，因此本地不需要额外 CORS 代理配置。停止前端后执行 Pop-Location 或直接关闭终端。

## 7. 启动后验收

### 7.1 API 健康检查

使用 curl.exe，避免 PowerShell 的 curl 别名差异：

~~~powershell
curl.exe --noproxy "*" -fsS http://127.0.0.1:8080/healthz
curl.exe --noproxy "*" -fsS http://127.0.0.1:8080/readyz
curl.exe --noproxy "*" -fsS http://127.0.0.1:8080/api/v1/ping
~~~

预期分别包含 "status":"ok"、就绪状态 ok 和 "ping":"pong"。其中 /readyz 失败时优先检查 MySQL 登录信息和迁移状态。

### 7.2 浏览器验收路径

1. 打开首页，确认示例文章列表可见。
2. 打开任意文章，确认 Markdown、代码高亮、目录和评论区域可渲染。
3. 访问 /login 登录普通账号，访问 /studio 验证个人文章工作台。
4. 访问 /admin/login 登录管理员，访问 /admin 验证用户/评论管理。
5. 上传图片后确认浏览器能访问 /uploads/<文件名>。

## 8. 常用命令

| 目的 | PowerShell 命令 |
| --- | --- |
| 迁移状态 | go run ./cmd/migrate -cmd status |
| 执行迁移 | go run ./cmd/migrate -cmd up |
| 回滚最近迁移 | go run ./cmd/migrate -cmd down（先在副本验证） |
| 导入示例文章 | go run ./cmd/seed |
| 渲染检查 | go run ./cmd/seed -dry-run |
| 后端测试 | go test ./... |
| 静态检查 | go vet ./... |
| 构建后端 | go build ./... |
| 前端类型检查 | Push-Location web; npm.cmd run typecheck; Pop-Location |
| 前端生产构建 | Push-Location web; npm.cmd run build; Pop-Location |
| 重新生成高亮 CSS | go run ./cmd/gencss -out web/src/styles/chroma.css |

仓库的 Makefile 提供同名快捷命令，但当前环境未检测到 GNU Make；Windows 下优先使用本表的 go / npm.cmd 命令。

## 9. 常见问题

### Go 报 go.mod requires go >= 1.26.0

当前机器只有 Go 1.24.5。安装并切换到 Go 1.26.0+，然后重新打开终端确认 go version。不要通过修改 go.mod 临时降级项目基线。

### Go 尝试下载工具链并报 Access is denied

这是自动工具链下载目录权限问题，不是项目代码错误。安装本机 Go 1.26 后可绕过该下载；如果仍需自动下载，需修复当前用户对 GOPATH（目前是 C:\Users\pc\go）的写权限。

### npm 报禁止运行 npm.ps1

改用 npm.cmd：

~~~powershell
& npm.cmd run build
~~~

### /readyz 返回 db unavailable 或迁移连接失败

依次检查：Get-Service MySQL80 是否为 Running；3306 是否被监听；.env 的用户名、密码、库名是否正确；再执行 go run ./cmd/migrate -cmd status。当前环境已实测 .env 的原有凭据被 MySQL 拒绝，需要先修正账号密码。

### 前端打开但接口 404 或网络错误

确认 Go API 在 8080 运行，并从 http://localhost:5173 访问前端。不要把前端直接用文件方式打开，也不要把 API 地址改成生产域名；代理规则位于 web/vite.config.ts。

### 图片上传失败

确认 data/uploads 可创建且 MEDIA_DIR 没有拼写错误。后端会限制单文件大小（默认 5 MB）和总配额（默认 2 GB），并拒绝 SVG。

## 10. 与线上部署的边界

本地流程不启动 Caddy、不使用 Docker、不暴露公网端口。线上部署请阅读 docs/deploy.md：它使用 Docker Compose、Caddy、独立 MySQL 容器和 HTTPS，不能把其中的 DATABASE_URL=mysql:3306 直接复制到本地 Windows 环境。

