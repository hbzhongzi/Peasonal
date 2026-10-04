// Command boot 创建博客的首个管理员账号。
//
// 设计取舍：管理员不走「开放注册」，而是由运维显式引导创建。
// 用环境变量或命令行参数提供邮箱/密码；库里已有管理员时拒绝重复创建。
//
// 用法：
//
//	go run ./cmd/boot                       # 用默认 admin@example.com / admin123456
//	ADMIN_EMAIL=me@x.com ADMIN_PASSWORD=xxx go run ./cmd/boot
//	go run ./cmd/boot -email me@x.com -password xxx
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"personal-blog/internal/config"
	"personal-blog/internal/repository"
	"personal-blog/internal/service"
)

func main() {
	envFile := flag.String("env", ".env", "配置文件路径，不存在则只读环境变量")
	email := flag.String("email", lookupEnv("ADMIN_EMAIL", "admin@example.com"), "管理员邮箱")
	password := flag.String("password", lookupEnv("ADMIN_PASSWORD", "admin123456"), "管理员密码")
	display := flag.String("name", "站长", "显示名")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	db, err := repository.Open(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "数据库不可用: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	queries := repository.New(db)
	authSvc := service.NewAuthService(queries, cfg.JWTSecret)
	ctx := context.Background()

	// 用默认值时要提醒，避免部署后忘了改密码
	if os.Getenv("ADMIN_EMAIL") == "" || os.Getenv("ADMIN_PASSWORD") == "" {
		slog.Warn("使用了默认管理员凭据，生产环境请务必通过环境变量指定",
			"email", *email)
	}

	user, err := authSvc.CreateFirstAdmin(ctx, *email, *password, *display)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建管理员失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 已创建管理员：%s（显示名：%s）\n", user.Email, user.DisplayName)
}

func lookupEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
