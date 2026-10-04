# 部署文档 · 个人博客（Docker Compose + Caddy）

> **文档状态说明（2026-10-04）**：本文是早期部署方案记录。当前可直接执行的线上手册是 [`docker-production-deploy.md`](./docker-production-deploy.md)，配套配置已落在仓库根目录的 `Dockerfile`、`docker-compose.prod.yml`、`Caddyfile` 和 `.dockerignore`。本文中的旧版配置片段不要直接复制到生产环境。

> 适用：把本博客部署到一台 Linux 云主机（已确认 **4 核 4G**），对外只暴露 80/443，数据库与后端都在内网。
> 本文是**早期方案记录**，不是当前生产配置的唯一入口。当前可执行手册与仓库根配置见 [`docker-production-deploy.md`](./docker-production-deploy.md)。
> 关联设计：技术方案 §9、附录 C（容量/MySQL 参数）、附录 D（Compose 方案）。

---

## 0. 与技术方案的重要差异（必读）

技术方案 §291 原设想「**Go 单二进制 embed 前端 dist**」。但**当前代码并未实现 embed**：`cmd/server` 只服务 `/api/*`、`/uploads/*`、`/healthz`、`/readyz`，没有 `//go:embed`、也没有 SPA 静态处理。

因此本部署采用更稳妥、也更简单的形态：

- **前端静态资源由 Caddy `file_server` 直接出**（构建产物 `web/dist`），并做 SPA history 回退（`try_files {path} /index.html`）。
- **Go 只负责 API、上传写入、健康检查**。
- 这与附录 D 的「Caddy 是唯一入站口、负责反代与静态/上传直出」完全一致，只是把"前端 dist 放在 Go 镜像里"换成了"前端 dist 放在 Caddy 挂载卷里"。**不需要改任何 Go 代码即可上线。**

> 如果你后续想要"单二进制"形态，只需在 `cmd/server` 里 `//go:embed web/dist` 并加一个 SPA handler，再把 dist 打进镜像即可——本文保留这一选项在 §12 备注，但默认按 Caddy 服务模式部署。

---

## 1. 拓扑

```
            ┌────────────────────────────────────────────┐
  浏览器 ──▶ │  单台主机 · Docker Compose                │
            │                                            │
            │   :80/:443  ┌──────────┐                   │
            │  唯一对外 ──▶│  caddy   │                   │
            │             │ (自动HTTPS)│                  │
            │             └──┬─────┬──┘                  │
            │      /api/*   │     │ /uploads/* 与 /(SPA) │
            │               │     │                      │
            │        reverse │     │ file_server          │
            │               ▼     ▼                      │
            │        ┌──────────┐ ┌────────────┐        │
            │        │  app     │ │ uploads 卷  │        │  ← app 写、caddy 读（共享卷）
            │        │ :8080    │ └────────────┘        │
            │        └────┬─────┘                       │
            │             │ tcp(mysql:3306)             │
            │        ┌────▼─────┐                       │
            │        │  mysql   │ (仅内网，不映射宿主机) │
            │        │  :3306   │                       │
            │        └──────────┘                       │
            └────────────────────────────────────────────┘
```

原则：**`caddy` 是唯一入站口**，`app` 与 `mysql` 都在内部 bridge 网络，**不映射宿主机端口**。数据库不暴露公网是最省事也最有效的一道安全闸。

---

## 2. 服务器与环境准备

| 项 | 要求 |
| --- | --- |
| 系统 | 任意装了 Docker 的 Linux（Debian/Ubuntu 推荐） |
| 规格 | **4 核 4G**（MySQL 限 2G、app 限 256M，余量充足） |
| 域名 | 一个已解析到本机公网 IP 的域名（如 `blog.example.com`） |
| 备案 | **国内云主机必须已完成 ICP 备案**，否则 80/443 可能被封、Let's Encrypt 签发失败 |
| 防火墙 | 只放行进 22（SSH）、80、443；3306/8080 一律不对外 |
| Docker | Docker Engine ≥ 24 + Docker Compose v2（`docker compose` 子命令） |

安装 Docker（以 Debian/Ubuntu 为例）：

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # 退出重登后免 sudo
docker compose version          # 确认 v2 可用
```

DNS：把 `blog.example.com` 的 A 记录指向服务器公网 IP。部署前先用 `ping blog.example.com` 确认解析已生效。

---

## 3. 部署目录约定

在服务器上建一个部署目录（建议用 git 拉取仓库，便于升级）：

```bash
git clone <你的仓库> /opt/blog
cd /opt/blog
```

需要的文件（全部在仓库根，§6 给出全文）：

```
/opt/blog/
├── Dockerfile            # 构建 app / migrate 二进制镜像（distroless）
├── docker-compose.yml    # caddy + app + mysql + 一次性 migrate
├── Caddyfile            # 自动 HTTPS + 反代 + 静态/SPA + uploads 直出
├── .dockerignore
├── .env                 # 生产环境变量（不入库！已在 .gitignore）
├── web/dist/            # 前端构建产物（下面会构建）
└── data/uploads/        # 运行时图片（由 docker 卷持久化，见下）
```

> `web/dist` 在服务器上构建（§5），也可在 CI 构建后传上来。本文以"服务器上构建"为主线。

---

## 4. 环境变量（生产 `.env` 模板）

把仓库里的 `.env.example` 复制为 `.env`，**并按下表改成生产值**。所有服务（app / migrate / boot）共用同一份 `.env`（注意 `config.Load` 要求 `JWT_SECRET` 必填，连 migrate 也需要它）。

```dotenv
# ===== 应用 =====
APP_ENV=production
HTTP_ADDR=:8080

# ===== 数据库（注意 host 是 compose 里的服务名 mysql，不是 127.0.0.1）=====
# parseTime / loc / UTC / charset 程序会自动补齐，但写全更直观
DATABASE_URL=root:你的强密码@tcp(mysql:3306)/blog?parseTime=true&loc=UTC&charset=utf8mb4
DB_MAX_OPEN=25
DB_MAX_IDLE=5
DB_MAX_LIFETIME=30m

# ===== 认证（JWT 至少 32 字节随机串）=====
# 生成：openssl rand -base64 48
JWT_SECRET=在这里填至少48字节的随机字符串不要留空

# ===== 媒体（本地磁盘；/srv/uploads 是容器内挂载点）=====
MEDIA_DIR=/srv/uploads
MEDIA_URL_PREFIX=/uploads
MEDIA_MAX_UPLOAD_MB=5
MEDIA_QUOTA_MB=2048

# ===== 站点 =====
BASE_URL=https://blog.example.com
# 同域部署时浏览器不发 Origin，CORS 不会触发；保留站点域名即可
CORS_ORIGINS=https://blog.example.com
RATE_LIMIT_RPS=10

# ===== 首个管理员（仅 cmd/boot 使用；用完可从 .env 删掉）=====
ADMIN_EMAIL=you@example.com
ADMIN_PASSWORD=给管理员设一个强密码
```

`.env` 一定加入 `.gitignore`，**绝不入库**。MySQL 的 root 密码也通过 compose 的环境变量传入（见 §6 的 `MYSQL_ROOT_PASSWORD`，与上面 `DATABASE_URL` 里的密码保持一致）。

---

## 5. 构建

### 5.1 前端（产出 `web/dist`）

```bash
cd /opt/blog/web
npm ci            # 或 npm install
npm run build     # 产物在 web/dist（index.html + assets/）
```

若构建机与运行机不同，把 `web/dist` 整个目录传到服务器 `/opt/blog/web/dist`。

### 5.2 后端镜像

`Dockerfile`（多阶段，distroless，静态二进制）见 §6。直接：

```bash
cd /opt/blog
docker compose build app migrate
```

这会得到 `app` 镜像（含 server 二进制）和 `migrate` 镜像（含迁移二进制）。**前端 dist 不进镜像**，由 Caddy 挂载读取。

---

## 6. 配套配置文件（全文）

### 6.1 `Dockerfile`

```dockerfile
# ---- 构建阶段 ----
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 静态链接，便于放进 distroless
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

# ---- 运行阶段 ----
FROM gcr.io/distroless/static-debian12 AS app
WORKDIR /
COPY --from=build /out/server /server
# 上传目录：distroless 默认以 nonroot(uid 65532) 运行，提前建好并赋权
RUN mkdir -p /srv/uploads && chown 65532:65532 /srv/uploads
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/server"]

# migrate 用同一份构建产物，单独一个 target 方便 `docker compose run --rm migrate`
FROM app AS migrate
ENTRYPOINT ["/migrate", "-cmd", "up"]
```

> 若拉不到 `gcr.io/distroless`，可换 `mirror.gcr.io/distroless/static-debian12` 或 `alpine` + 静态二进制（alpine 需 `ca-certificates`，且注意 nonroot 权限）。

### 6.2 `docker-compose.yml`

```yaml
services:
  caddy:
    image: caddy:2
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./web/dist:/srv/dist:ro          # 前端构建产物
      - uploads:/srv/uploads:ro          # 与 app 共享，只读给 caddy 出图
      - caddy-data:/data
      - caddy-config:/config
    depends_on:
      - app
    restart: unless-stopped

  app:
    build:
      context: .
      target: app
    env_file: .env
    volumes:
      - uploads:/srv/uploads             # 读写：app 写图
    # 不映射 8080 到宿主机；只对内网
    expose:
      - "8080"
    depends_on:
      mysql:
        condition: service_healthy
    deploy:
      resources:
        limits:
          memory: 256M
    restart: unless-stopped

  mysql:
    image: mysql:8.0
    env_file: .env                       # 复用其中的强密码（见下 MYSQL_ROOT_PASSWORD）
    environment:
      MYSQL_ROOT_PASSWORD: "${MYSQL_ROOT_PASSWORD}"
      MYSQL_DATABASE: blog
    command:
      - --ngram_token_size=2
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_0900_ai_ci
    volumes:
      - mysql-data:/var/lib/mysql
    # 不映射 3306 到宿主机
    expose:
      - "3306"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-p${MYSQL_ROOT_PASSWORD}"]
      interval: 10s
      timeout: 5s
      retries: 10
    deploy:
      resources:
        limits:
          memory: 2G
    restart: unless-stopped

  # 一次性迁移任务（不在 up 时自动跑，避免多实例竞态）
  migrate:
    build:
      context: .
      target: migrate
    env_file: .env
    depends_on:
      mysql:
        condition: service_healthy
    restart: "no"

volumes:
  mysql-data:
  uploads:
  caddy-data:
  caddy-config:
```

> `MYSQL_ROOT_PASSWORD` 需在 `.env` 中**额外加一行**（compose 的 `env_file` 会把 `.env` 注入各服务环境），例如：
> ```dotenv
> MYSQL_ROOT_PASSWORD=与DATABASE_URL里一致的强密码
> ```
> 它与 `DATABASE_URL` 里的密码必须相同，否则 app 连不上库。

### 6.3 `Caddyfile`

```caddyfile
blog.example.com {
    encode gzip

    # 已上传图片：app 写入共享卷，caddy 直出，不经过 Go（零运行时开销）
    handle /uploads/* {
        root * /srv/uploads
        file_server
    }

    # API 反代到 Go
    handle /api/* {
        reverse_proxy app:8080
    }

    # 前端 SPA：静态资源 + history 回退
    handle {
        root * /srv/dist
        try_files {path} /index.html
        file_server
    }
}
```

> 若 Caddy 与 app 都注册了 `/uploads/*`（代码里也有一个 Go 静态处理器），**Caddy 的 `handle` 优先级更高**，请求不会进到 Go。本地开发时（无 Caddy）则由 Go 兜底直出，行为一致。

### 6.4 `.dockerignore`

```gitignore
.git
web/node_modules
web/dist
data
*.exe
.env
```

---

## 7. 首次上线流程

```bash
cd /opt/blog

# 1) 准备 .env（见 §4），确认 DATABASE_URL / JWT_SECRET / MYSQL_ROOT_PASSWORD 都填好

# 2) 构建前端 + 镜像
cd web && npm run build && cd ..
docker compose build app migrate

# 3) 起数据库（等它 healthy）
docker compose up -d mysql
docker compose ps            # 看到 mysql 为 healthy 再继续

# 4) 跑迁移（建表 + 索引，含 ngram FULLTEXT）
docker compose run --rm migrate

# 5) 建首个管理员（用 ADMIN_EMAIL / ADMIN_PASSWORD）
docker compose run --rm --entrypoint "" app \
  /server -env /dev/null boot 2>/dev/null || true
# ↑ 上面这行若不便，可临时改 app 镜像 entrypoint 跑 boot；更简单的做法见下：
docker compose run --rm -e ADMIN_EMAIL=you@example.com -e ADMIN_PASSWORD=强密码 \
  --entrypoint /server app -env .env boot
# 注：cmd/boot 读取 .env 中的 ADMIN_EMAIL/ADMIN_PASSWORD，库里已有管理员则跳过。

# 6) 起全部服务
docker compose up -d

# 7) 冒烟测试（见 §8）
```

> 第 5 步的 boot 入口：因为 `app` 镜像的 ENTRYPOINT 是 `/server`（server 不支持 `boot` 子命令），实际应让 boot 用 **migrate 同款的构建**或单独一个 boot target。最省事的做法：把 `cmd/boot` 也编进镜像（在 Dockerfile 里 `go build -o /out/boot ./cmd/boot`，并加一个 `boot` target，`ENTRYPOINT ["/boot"]`），然后 `docker compose run --rm boot`。本文推荐在 Dockerfile 增加 `boot` target，避免临时改 entrypoint 的脆弱写法。

---

## 8. 健康检查与冒烟测试

容器就绪探针（app 自身提供）：

- `GET /healthz` → `{"status":"ok"}`（只验证进程存活）
- `GET /readyz` → 验证 DB 连通（K8s/Docker 健康检查用）

手工冒烟：

```bash
# 在服务器上
curl -fsS https://blog.example.com/api/v1/ping && echo "  API OK"
curl -fsS https://blog.example.com/              | grep -q '<div id="root">' && echo "  SPA OK"
# 看容器日志
docker compose logs -f app
docker compose logs -f caddy
```

浏览器访问 `https://blog.example.com`：应自动跳 HTTPS、能看到首页文章；登录 `/login` 后进入 `/studio` 可写文章、传图；图片 URL 形如 `https://blog.example.com/uploads/...` 能直接打开。

---

## 9. 数据库初始化要点

- **字符集**：`utf8mb4` / `utf8mb4_0900_ai_ci`（compose 的 `command` 已带；建库语句同 `.env.example` 注释）。
- **中文全文检索**：`ngram_token_size=2` 是**只读启动参数**，只能写在 mysql 的 `command` 里（已写）。改完必须**重启 MySQL 并重建 FULLTEXT 索引**才生效。本项目迁移里已包含 `FULLTEXT ... WITH PARSER ngram`，首次 `migrate` 即建好。
- 迁移是**追加式**：已执行的迁移不可改写；改 schema 一律追加新迁移。
- 回滚：`docker compose run --rm migrate -cmd down`（**未逐个实测 `down`，生产慎用**，见 §11）。

---

## 10. 备份（必须异地）

原则：**备份不能与数据放同一块盘**。每日把 `mysql-data` 与 `uploads` 导出，再 `rsync`/对象存储到另一台机器或云。

```bash
#!/usr/bin/env bash
# backup.sh —— 每天 cron 执行，产物留在本地后由 rsync 异地
set -euo pipefail
DATE=$(date +%F)
DEST=/opt/blog/backups/$DATE
mkdir -p "$DEST"

# 1) 数据库 dump（--single-transaction 不锁表）
docker compose exec -T mysql \
  mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --routines blog \
  > "$DEST/blog.sql"

# 2) 图片卷打包
docker run --rm -v blog_uploads:/data -v "$DEST":/backup alpine \
  tar czf /backup/uploads.tar.gz -C /data .

# 3) 异地（示例：rsync 到备份机；也可 rclone 到对象存储）
# rsync -az "$DEST" backup@backup-host:/srv/blog-backups/
echo "backup done: $DEST"
```

恢复：先把 `blog.sql` 导入新建的库，再把 `uploads.tar.gz` 解回 `uploads` 卷。

---

## 11. 升级与回滚

**升级**：

```bash
cd /opt/blog
git pull
cd web && npm run build && cd ..
docker compose build app migrate          # 建议给镜像打 git sha 标签
docker compose up -d app                  # 滚动替换 app
curl -fsS https://blog.example.com/api/v1/ping   # 确认 /readyz 通过
```

**回滚**：镜像打 tag（如 `app:sha-abc`、`app:sha-def`），回滚时改 `image: app:sha-abc` 后 `docker compose up -d app`。数据库**不随意回滚**（迁移向下兼容需单独评估）。

**迁移**：每次发版若含新迁移，先 `docker compose run --rm migrate` 再起 `app`。`down` 仅作应急，未在生产验证，使用前务必在预发环境演练。

---

## 12. 排错指南

| 现象 | 原因 / 排查 |
| --- | --- |
| 上传报权限错误（`permission denied`） | 最常见坑：app 以 nonroot(uid 65532) 运行，`uploads` 卷不可写。Dockerfile 已 `chown 65532 /srv/uploads`；若用 bind mount 的 `./data/uploads`，需 `chown -R 65532 ./data/uploads`。 |
| Caddy 证书签发失败 / 一直 staging | 80 端口未放行、域名未备案（国内）、DNS 未生效。先 `curl -I http://blog.example.com` 确认 80 通。 |
| app 起不来，`DATABASE_URL` 报错 | `mysql` 服务名解析不到：确认 app 与 mysql 在同一 compose 网络、DATABASE_URL 用的是 `mysql:3306` 而非 `127.0.0.1`。 |
| `/readyz` 一直失败 | MySQL 还没 healthy，或密码不一致（`MYSQL_ROOT_PASSWORD` 与 `DATABASE_URL` 必须相同）。 |
| 深链接 404（如 `/posts/xxx` 刷新后 404） | Caddy 的 SPA 回退没生效：确认 `try_files {path} /index.html` 写在与 `handle /api/*` 平级的 `handle {}` 里。 |
| 图片 403/404 | `/uploads/*` 被 Caddy 优先级更高的 `handle` 截走但卷没挂对；确认 `uploads` 卷同时挂到 app(rw) 与 caddy(ro) 且路径都是 `/srv/uploads`。 |
| JWT 全部失效、登录即掉线 | `JWT_SECRET` 改了或每次重启都随机——本程序**禁止**运行时兜底随机密钥；确认 `.env` 里 `JWT_SECRET` 固定且各容器一致。 |

---

## 13. 安全清单（上线前逐项确认）

- [ ] `.env` 在 `.gitignore`，**未入库**；`JWT_SECRET` 为 ≥48 字节随机串
- [ ] 防火墙只开 22/80/443；3306、8080 不对公网
- [ ] MySQL root 密码强且唯一；`DATABASE_URL` 与之匹配
- [ ] Caddy 自动 HTTPS 生效，浏览器地址栏有锁；建议加 HSTS
- [ ] `uploads` 卷属主对齐 nonroot，写入正常
- [ ] 每日备份脚本已配置并**异地**落盘；做过一次恢复演练
- [ ] 管理员账号已建、默认密码已改
- [ ] 国内主机已完成 ICP 备案

---

## 14. 与技术方案的偏差小结（实现记录）

| 技术方案原设 | 本部署实际 | 处理 |
| --- | --- | --- |
| Go embed 前端 dist（§291） | 前端 dist 由 Caddy `file_server` 直出 | 不改 Go 代码即可上线；单二进制选项保留（§0 备注） |
| Dockerfile / docker-compose 标「M5 暂缓未建」 | 本文给出**完整可用**的 Dockerfile / compose / Caddyfile | 直接可用，无需再从零设计 |
| 迁移随 app 启动跑 | 独立一次性 `migrate` 容器（`docker compose run --rm migrate`） | 避免多实例竞态，失败可见 |

> 以上偏差属实现取舍，不违反架构原则（Caddy 唯一入站、DB 不暴露、上传读公开写受限均保持）。待 M5 正式收尾时，可把 Dockerfile/compose/Caddyfile 落到仓库根，并在技术方案附录 D 标注「已实现，见 deploy.md」。
