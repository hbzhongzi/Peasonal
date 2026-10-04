# syntax=docker/dockerfile:1

# ---------- 前端构建 ----------
FROM node:22-bookworm-slim AS web-build
WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# ---------- Go 构建 ----------
FROM golang:1.26-bookworm AS go-build
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/boot ./cmd/boot \
    && mkdir -p /out/uploads \
    && touch /out/uploads/.keep

# ---------- Go 运行时 ----------
# 使用 distroless nonroot；上传目录通过 COPY --chown 预先创建，
# 首次挂载 named volume 时会继承该目录权限。
FROM gcr.io/distroless/static-debian12:nonroot AS app
WORKDIR /app
COPY --from=go-build /out/server /server
COPY --from=go-build /out/migrate /migrate
COPY --from=go-build /out/boot /boot
COPY --from=go-build /src/internal/repository/migrations /app/internal/repository/migrations
COPY --from=go-build --chown=65532:65532 /out/uploads/.keep /srv/uploads/.keep
USER 65532:65532
ENTRYPOINT ["/server"]

FROM app AS migrate
ENTRYPOINT ["/migrate"]

FROM app AS boot
ENTRYPOINT ["/boot"]

# ---------- Caddy + 前端静态资源 ----------
FROM caddy:2-alpine AS caddy
COPY --from=web-build /src/web/dist /srv/dist
