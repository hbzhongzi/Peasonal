// Command server 是服务入口：读配置 → 建依赖 → 起服务 → 优雅退出。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"personal-blog/internal/config"
	"personal-blog/internal/repository"
	httpapi "personal-blog/internal/transport/http"
)

func main() {
	envFile := flag.String("env", ".env", "配置文件路径，不存在则只读取环境变量")
	flag.Parse()

	cfg, err := config.Load(*envFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	db, err := repository.Open(cfg)
	if err != nil {
		logger.Error("数据库不可用", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 媒体目录提前建好，避免第一次上传时才发现没权限
	if err := os.MkdirAll(cfg.MediaDir, 0o755); err != nil {
		logger.Error("创建媒体目录失败", "dir", cfg.MediaDir, "error", err)
		os.Exit(1)
	}
	absMedia, _ := filepath.Abs(cfg.MediaDir)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpapi.NewRouter(httpapi.Deps{Config: cfg, Logger: logger, DB: db}),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// SIGTERM/SIGINT → ctx 取消 → 触发优雅退出
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("服务启动",
			"addr", cfg.HTTPAddr,
			"env", cfg.AppEnv,
			"media_dir", absMedia,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("服务异常退出", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	logger.Info("收到退出信号，等待在途请求结束", "timeout", cfg.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("优雅退出失败，强制关闭", "error", err)
	}
	logger.Info("服务已停止")
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	if cfg.IsProduction() {
		opts.Level = slog.LevelInfo
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	// 开发环境用文本格式，便于人眼阅读
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
