.PHONY: help tidy fmt vet build run seed css migrate migrate-down migrate-status sqlc dev-api dev-web test boot

GO ?= go
NPM ?= npm

help: ## 显示可用命令
	@echo "可用命令："
	@echo "  make run            启动后端服务 (:8080)"
	@echo "  make migrate        执行数据库迁移（up）"
	@echo "  make seed           导入 testdata 下的示例文章"
	@echo "  make seed-dry       只渲染示例文章、打印结果，不连数据库"
	@echo "  make css            重新生成代码高亮 CSS"
	@echo "  make sqlc           由 queries/*.sql 生成 Go 代码（需先装 sqlc）"
	@echo "  make dev-web        启动前端开发服务器 (:5173)"
	@echo "  make build          构建全部二进制到 bin/"
	@echo "  make web-build      构建前端到 web/dist"
	@echo "  make test           跑 Go 测试"
	@echo ""
	@echo "注意：本机 MySQL 8.0 已作为 Windows 服务运行，本地开发不需要 docker。"

tidy: ## 整理依赖
	$(GO) mod tidy

fmt: ## 格式化
	$(GO) fmt ./...

vet: ## 静态检查
	$(GO) vet ./...

build: ## 构建全部二进制到 bin/
	$(GO) build -o bin/ ./cmd/server
	$(GO) build -o bin/ ./cmd/migrate
	$(GO) build -o bin/ ./cmd/seed

seed: ## 导入 testdata/posts/*.md（按 slug 判重，可反复跑）
	$(GO) run ./cmd/seed

seed-dry: ## 只渲染并打印结果，不连数据库
	$(GO) run ./cmd/seed -dry-run

css: ## 由 chroma 重新生成代码高亮 CSS
	$(GO) run ./cmd/gencss -out web/src/styles/chroma.css

web-build: ## 构建前端
	cd web && $(NPM) run build

run: ## 启动后端服务
	$(GO) run ./cmd/server

migrate: ## 执行迁移
	$(GO) run ./cmd/migrate -cmd up

migrate-down: ## 回滚最近一个迁移
	$(GO) run ./cmd/migrate -cmd down

migrate-status: ## 查看迁移状态
	$(GO) run ./cmd/migrate -cmd status

sqlc: ## 生成类型安全的数据访问代码
	sqlc generate

dev-web: ## 启动前端开发服务器
	cd web && $(NPM) run dev

test: ## 跑测试
	$(GO) test ./...

boot: ## 创建首个管理员（库里已有则跳过）
	$(GO) run ./cmd/boot
