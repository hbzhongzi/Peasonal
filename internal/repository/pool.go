// Package repository 是持久化层。所有 SQL 只出现在这里，上层不感知数据库。
//
// 注意：db.go / models.go / *.sql.go 由 sqlc 生成，**不要手写也不要改**。
// 手写代码放在别的文件里（如此文件），否则会被 sqlc generate 覆盖。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // 注册 mysql 驱动

	"personal-blog/internal/config"
)

// Open 建立数据库连接池并验证连通性。
//
// 连接池必须显式配置：默认的无上限连接会在流量上来时把 MySQL 拖垮
// （4C4G 机器上尤其明显，见设计文档附录 C）。
func Open(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpen)
	db.SetMaxIdleConns(cfg.DBMaxIdle)
	db.SetConnMaxLifetime(cfg.DBMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	return db, nil
}
