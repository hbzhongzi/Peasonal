// Command migrate 执行数据库迁移。
//
// 单独一个命令而不是放在服务启动时跑，是为了避免多实例竞态，
// 也让迁移失败能在部署日志里一眼看到（见设计文档 §7.6）。
//
// 用法：
//
//	go run ./cmd/migrate -cmd up
//	go run ./cmd/migrate -cmd down     # 回滚最近一个迁移
//	go run ./cmd/migrate -cmd status
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"

	"personal-blog/internal/config"
)

func main() {
	cmd := flag.String("cmd", "up", "up | down | status | reset")
	dir := flag.String("dir", "internal/repository/migrations", "迁移文件目录")
	envFile := flag.String("env", ".env", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("mysql", cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// MySQL 的 DDL 会隐式提交，迁移文件里全部标注了 NO TRANSACTION
	if err := goose.SetDialect("mysql"); err != nil {
		fmt.Fprintf(os.Stderr, "设置方言失败: %v\n", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	log.Info("执行迁移", "command", *cmd, "dir", *dir)

	var migrateErr error
	switch *cmd {
	case "up":
		migrateErr = goose.Up(db, *dir)
	case "down":
		migrateErr = goose.Down(db, *dir)
	case "status":
		migrateErr = goose.Status(db, *dir)
	case "reset":
		migrateErr = goose.Reset(db, *dir)
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", *cmd)
		os.Exit(1)
	}

	if migrateErr != nil {
		log.Error("迁移失败", "error", migrateErr)
		os.Exit(1)
	}
	log.Info("迁移完成")
}
