// Package httpapi 是 HTTP 传输层：路由、中间件、DTO 转换。
//
// 这一层只做三件事：解析请求参数、调用 service、把结果序列化成 JSON。
// 业务规则一律不放这里。
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"personal-blog/internal/config"
	"personal-blog/internal/pkg/markdown"
	"personal-blog/internal/pkg/media"
	"personal-blog/internal/repository"
	"personal-blog/internal/service"
	"personal-blog/internal/transport/http/middleware"
)

// Deps 是传输层需要的全部依赖，由 cmd/server 装配后注入。
type Deps struct {
	Config config.Config
	Logger *slog.Logger
	DB     *sql.DB
}

// NewRouter 构建完整路由树。
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	// 中间件顺序有讲究：RequestID 要在最前面，后续日志才能带上它
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(d.Logger))

	// 图片文件落本地磁盘，这里只做静态直出（任何人可读；上传/删除另有鉴权）
	r.Handle("/uploads/*", http.StripPrefix("/uploads/",
		http.FileServer(http.Dir(d.Config.MediaDir))))

	// 存活探针：只要进程还在就返回 200，不查依赖
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// 就绪探针：查库连通性，用于判断能否接收流量
	r.Get("/readyz", handleReadyz(d.DB))

	queries := repository.New(d.DB)

	// 共享依赖
	auther := middleware.NewAuther(d.Config.JWTSecret)
	md := markdown.New()
	store, err := media.NewStore(d.Config.MediaDir, d.Config.MediaURLPrefix)
	if err != nil {
		d.Logger.Error("媒体目录初始化失败", "err", err)
		// 不影响启动，但上传会失败；记录后继续
	}

	// service 层
	postSvc := service.NewPostService(queries)
	authSvc := service.NewAuthService(queries, d.Config.JWTSecret)
	adminPostSvc := service.NewAdminPostService(queries, md)
	commentSvc := service.NewCommentService(queries, d.Config.JWTSecret) // pepper 复用 JWT 密钥
	mediaSvc := service.NewMediaService(queries, store, d.Config.MediaQuotaMB, d.Config.MediaURLPrefix)

	prod := d.Config.IsProduction()

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"ping": "pong"})
		})

		// 公开接口
		NewPostHandler(postSvc, d.Logger).Register(r)
		NewCommentHandler(commentSvc, d.Logger).RegisterPublic(r)
		NewSiteHandler(authSvc, d.Logger).Register(r) // /site/profile：站长公开资料，匿名可读

		// 登录/刷新/登出/注册（无需鉴权）
		NewAuthHandler(authSvc, auther, d.Logger, prod).Register(r)

		// 受保护的管理组：各 handler 自行声明所需权限级别
		//   RequirePublish：发帖/管理（文章、评论、媒体管理、用户管理）
		//   RequireUpload：上传图片（RequirePublish 的子集，由媒体 handler 单独包裹）
		r.Group(func(r chi.Router) {
			NewAuthHandler(authSvc, auther, d.Logger, prod).RegisterAuthed(r) // /admin/me + /admin/users
			NewAdminPostHandler(adminPostSvc, d.Logger).Register(r, auther)   // 全部 RequirePublish
			NewMediaHandler(mediaSvc, d.Logger, d.Config.MediaMaxUploadMB).Register(r, auther)
			NewCommentHandler(commentSvc, d.Logger).RegisterAdmin(r, auther)
		})
	})

	return r
}

func handleReadyz(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "db unavailable",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": body})
}
