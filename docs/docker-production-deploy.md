# Docker 线上服务器部署说明

> 适用项目：个人博客（Go 1.26 + React/Vite + MySQL 8.0）  
> 部署形态：单台 Linux 云服务器 + Docker Compose + Caddy 自动 HTTPS  
> 推荐规格：4 vCPU / 4 GB RAM  
> 本文是当前仓库的线上部署手册；本地 Windows 启动请看 [`local-deploy.md`](./local-deploy.md)。

## 1. 部署架构

```text
公网 80/443
    │
    ▼
 Caddy（TLS、静态前端、上传图片）
    │ /api/*
    ▼
 Go app :8080（仅 Compose 内网）
    │
    ▼
 MySQL :3306（仅 Compose 内网）
```

只有 Caddy 暴露 80/443；Go 和 MySQL 不映射宿主机端口。前端在镜像构建阶段由 Node 编译，Caddy 运行时直接提供 `web/dist`；Go 只处理 API、健康检查和上传写入。

仓库根目录已经提供以下线上配置：

| 文件 | 作用 |
| --- | --- |
| `Dockerfile` | 构建 `app`、`migrate`、`boot` 和 `caddy` 四个镜像目标 |
| `docker-compose.prod.yml` | MySQL、Go、Caddy 及运维命令容器 |
| `Caddyfile` | HTTPS、API 反代、上传文件和 SPA 回退 |
| `.dockerignore` | 排除本地密钥、依赖、运行时数据 |

## 2. 上线前提

- Linux 主机，建议 Debian 12 或 Ubuntu 22.04/24.04。
- Docker Engine 24+，Docker Compose v2。
- 域名 A/AAAA 记录已指向服务器公网 IP。
- 云安全组和主机防火墙仅放行 TCP `22`、`80`、`443`；不要开放 `3306`、`8080`。
- 国内服务器使用域名申请公网 HTTPS 前，确认域名备案和云厂商备案要求。
- 服务器有至少 10 GB 可用磁盘；图片增长主要消耗磁盘，不是内存。

## 3. 服务器初始化

以下以 Debian/Ubuntu 为例：

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl git ufw
curl -fsSL https://get.docker.com | sudo sh
sudo systemctl enable --now docker
sudo usermod -aG docker "$(id -un)"
```

重新登录 SSH 后确认：

```bash
docker version
docker compose version
```

配置防火墙：

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw --force enable
sudo ufw status
```

## 4. 获取代码和准备配置

建议部署到 `/opt/personal-blog`：

```bash
sudo mkdir -p /opt/personal-blog
sudo chown "$(id -un)":"$(id -gn)" /opt/personal-blog
git clone <仓库地址> /opt/personal-blog
cd /opt/personal-blog
```

复制生产配置：

```bash
cp .env.example .env
chmod 600 .env
```

编辑 `.env`。线上至少需要这些变量：

```dotenv
APP_ENV=production
HTTP_ADDR=:8080

# Compose 服务名必须是 mysql，不是 127.0.0.1
DATABASE_URL=blog_app:数据库密码@tcp(mysql:3306)/blog
DB_MAX_OPEN=25
DB_MAX_IDLE=5
DB_MAX_LIFETIME=30m

# 固定密钥，至少 32 字节；不要每次部署重新生成
JWT_SECRET=替换为随机长字符串

MEDIA_DIR=/srv/uploads
MEDIA_URL_PREFIX=/uploads
MEDIA_MAX_UPLOAD_MB=5
MEDIA_QUOTA_MB=2048

BASE_URL=https://blog.example.com
CORS_ORIGINS=https://blog.example.com
RATE_LIMIT_RPS=10

# MySQL 容器初始化变量；MYSQL_PASSWORD 必须和 DATABASE_URL 中的密码一致
MYSQL_ROOT_PASSWORD=替换为独立的 root 强密码
MYSQL_DATABASE=blog
MYSQL_USER=blog_app
MYSQL_PASSWORD=替换为数据库密码

# Caddy 自动签发证书
SITE_DOMAIN=blog.example.com
ACME_EMAIL=you@example.com
```

密码不要包含未转义的 `@`、`:`、`/` 等 DSN 保留字符；推荐使用十六进制随机字符串：

```bash
openssl rand -hex 32
```

确认 `DATABASE_URL` 中的用户、密码、数据库名分别与 `MYSQL_USER`、`MYSQL_PASSWORD`、`MYSQL_DATABASE` 一致。MySQL 数据卷已经初始化后，修改 `MYSQL_*` 不会自动修改已有账号，必须按第 10 节手工改库或恢复数据。

## 5. 检查域名和端口

部署前在本地确认 DNS：

```bash
dig +short blog.example.com
```

服务器上确认端口没有被其他 Web 服务占用：

```bash
sudo ss -lntp | grep -E ':(80|443)\b' || true
```

Caddy 只有在域名已经解析、80/443 可从公网访问时才能正常申请 Let's Encrypt 证书。首次证书申请失败时不要反复重启刷请求，先修正 DNS 或防火墙。

## 6. 构建镜像

在仓库根目录执行：

```bash
docker compose -f docker-compose.prod.yml build --pull app migrate boot caddy
```

构建内容：

- `node:22` 编译前端并生成 `web/dist`；
- `golang:1.26` 编译 `server`、`migrate`、`boot`；
- `caddy` 镜像携带前端静态文件；
- Go 运行镜像使用 distroless nonroot 用户 `65532`。

如果服务器无法访问 Docker Hub，可在可联网的构建机执行构建并推送镜像，再把 Compose 中的 `build` 改成对应 `image`；不要把 `.env` 打包进镜像。

## 7. 首次启动顺序

### 7.1 只启动数据库

```bash
docker compose -f docker-compose.prod.yml up -d mysql
docker compose -f docker-compose.prod.yml ps
```

继续前确认 `mysql` 状态为 `healthy`：

```bash
docker inspect --format '{{.State.Health.Status}}' "$(docker compose -f docker-compose.prod.yml ps -q mysql)"
```

### 7.2 执行数据库迁移

迁移是显式运维动作，不放进 `app` 启动命令，避免多实例同时执行 DDL：

```bash
docker compose -f docker-compose.prod.yml --profile ops run --rm migrate
docker compose -f docker-compose.prod.yml --profile ops run --rm migrate -cmd status
```

项目当前包含 25 个追加式迁移。已执行迁移不可改写，后续 schema 变更必须新增迁移文件。

### 7.3 创建首个管理员

管理员凭据通过一次性命令传入，不写入镜像。下面的邮箱和密码请替换：

```bash
docker compose -f docker-compose.prod.yml --profile ops run --rm \
  -e ADMIN_EMAIL=admin@example.com \
  -e ADMIN_PASSWORD='替换为强密码' \
  boot
```

如果数据库中已经存在管理员，命令会按程序规则拒绝重复创建；这不是故障。

### 7.4 启动应用和 Caddy

```bash
docker compose -f docker-compose.prod.yml up -d app caddy
docker compose -f docker-compose.prod.yml ps
```

查看日志：

```bash
docker compose -f docker-compose.prod.yml logs --tail=100 app
docker compose -f docker-compose.prod.yml logs --tail=100 caddy
```

## 8. 上线验收

先验证应用内部健康状态：

```bash
curl -fsS https://blog.example.com/healthz
curl -fsS https://blog.example.com/readyz
curl -fsS https://blog.example.com/api/v1/ping
```

再验证浏览器流程：

1. 打开 `https://blog.example.com`，确认首页返回前端页面。
2. 打开一篇文章，确认文章详情、代码高亮、评论区域正常。
3. 访问 `/admin/login`，使用首个管理员登录。
4. 访问 `/login` 或注册普通用户，确认 `/studio` 权限符合预期。
5. 上传图片并确认 `https://blog.example.com/uploads/<文件名>` 可访问。

深链接刷新测试：

```bash
curl -fsS https://blog.example.com/posts/example-slug | grep -q 'id="root"'
```

## 9. 日常更新

每次更新前先备份，再拉代码、重建镜像。包含迁移的版本必须先迁移、后替换应用：

```bash
cd /opt/personal-blog
./backup.sh
git pull --ff-only
docker compose -f docker-compose.prod.yml build --pull app migrate boot caddy
docker compose -f docker-compose.prod.yml --profile ops run --rm migrate
docker compose -f docker-compose.prod.yml up -d app caddy
curl -fsS https://blog.example.com/readyz
```

只改前端或 Go 代码也建议同时重建 `caddy` 和 `app`，保证静态资源与 API 版本一致。

## 10. 备份与恢复

备份必须包含数据库和上传图片，并复制到另一台机器或对象存储。可在 `/opt/personal-blog/backup.sh` 保存以下脚本：

```bash
#!/usr/bin/env bash
set -euo pipefail

cd /opt/personal-blog
backup_date="$(date +%F_%H%M%S)"
backup_dir="/opt/personal-blog/backups/${backup_date}"
mkdir -p "${backup_dir}"

# 密码从 mysql 服务自身的环境变量读取，不出现在命令行参数中。
docker compose -f docker-compose.prod.yml exec -T mysql \
  sh -c 'exec mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --quick --routines blog' \
  > "${backup_dir}/blog.sql"

docker run --rm \
  -v personal-blog-uploads:/data:ro \
  -v "${backup_dir}":/backup \
  alpine:3.20 \
  tar czf /backup/uploads.tar.gz -C /data .

echo "backup created: ${backup_dir}"
```

执行权限和手工测试：

```bash
chmod 700 /opt/personal-blog/backup.sh
/opt/personal-blog/backup.sh
```

恢复时先停止 `app`，将 SQL 导入同版本 MySQL，再恢复上传卷；恢复完成后执行迁移状态检查并启动服务。恢复操作不要直接覆盖生产卷，先在临时目录或备用主机演练。

## 11. 回滚原则

- 应用回滚：保留旧镜像 tag，改 Compose 的 `image` 或恢复旧代码后重新构建。
- 数据库迁移不可依赖应用镜像自动回滚；MySQL DDL 可能已隐式提交。
- 回滚前先确认旧应用能读取当前 schema；不确定时优先恢复数据库备份到独立环境验证。
- 不要执行 `docker compose down -v`，它会删除数据库、上传和 Caddy 数据卷。

## 12. 常见故障

| 现象 | 排查方向 |
| --- | --- |
| Caddy 无法签证书 | DNS 未生效、云安全组未放行 80/443、域名备案限制、已有程序占用端口；先看 `logs caddy` |
| `app` 退出并提示数据库连接失败 | `DATABASE_URL` 的 host 必须是 `mysql`；确认 `MYSQL_PASSWORD` 与 DSN 密码一致；检查 `mysql` 是否 healthy |
| `/readyz` 503 | MySQL 未就绪或账号权限错误；查看 `docker compose logs mysql` |
| 上传 `permission denied` | 检查 `uploads` named volume 是否由本 Compose 创建；不要把它改成 root-only 的宿主机 bind mount |
| 图片 404 | Caddy 使用了 `handle_path /uploads/*`，确认 Caddyfile 未改回 `handle`；确认 app 与 caddy 挂载同一个 `uploads` 卷 |
| 刷新文章链接 404 | Caddy 的 `try_files {path} /index.html` 必须保留 |
| 登录后令牌全部失效 | `JWT_SECRET` 被改动；生产环境必须固定且只保存在 `.env` |
| 修改 MYSQL_* 后无效 | MySQL 数据卷已经初始化；环境变量只在首次初始化数据库时生效，需手工修改账号或重建测试卷 |

## 13. 上线检查清单

- [ ] Go/前端镜像构建成功，使用当前仓库的 Dockerfile
- [ ] `.env` 权限为 `600`，没有提交到 Git
- [ ] `DATABASE_URL` 使用 `mysql` 服务名，不是 `localhost`
- [ ] `MYSQL_PASSWORD` 与 `DATABASE_URL` 中的应用密码一致
- [ ] `JWT_SECRET` 是固定的随机长字符串
- [ ] DNS 已生效，安全组和 UFW 只开放 22/80/443
- [ ] 25 个迁移已成功执行，`/readyz` 返回 200
- [ ] 管理员登录、普通用户登录、文章发布、图片上传均完成一次验证
- [ ] 数据库和 uploads 已完成一次恢复演练，并有异地副本
- [ ] 已配置磁盘、内存和容器日志监控

## 14. 与本地部署的区别

本地环境使用 Windows MySQL 服务、Go `go run` 和 Vite dev server；线上环境使用 Compose 内部 MySQL、构建后的 Go 二进制、Caddy 静态文件和自动 HTTPS。线上 `.env` 不能直接复制本地版本，尤其是 `DATABASE_URL` 的主机名、`APP_ENV`、`BASE_URL`、`CORS_ORIGINS` 和 MySQL 密钥。
