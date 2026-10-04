package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"log/slog"

	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/service"
)

// PostHandler 暴露文章的公开读接口。
//
// 这一层只做三件事：解析参数、调 service、序列化。业务规则一律不在这里。
type PostHandler struct {
	svc *service.PostService
	log *slog.Logger
}

func NewPostHandler(svc *service.PostService, log *slog.Logger) *PostHandler {
	return &PostHandler{svc: svc, log: log}
}

// Register 把文章相关的公开路由挂到 /api/v1 下。
func (h *PostHandler) Register(r chi.Router) {
	r.Get("/posts", h.list)
	r.Get("/posts/{slug}", h.detail)
	r.Get("/tags", h.tags)
	r.Get("/tags/{slug}/posts", h.listByTag)
	r.Get("/archive", h.archive)
	r.Get("/archive/{year}/{month}", h.listByMonth)
	r.Get("/posts/search", h.search)
	// 用户公开资料与其已发布文章：/blog/{publicID} 个人博客主页用
	r.Get("/users/{publicID}", h.authorProfile)
	r.Get("/users/{publicID}/posts", h.authorPosts)
}

// GET /api/v1/posts?page=1&page_size=10
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)

	res, err := h.svc.List(r.Context(), page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/posts/{slug}
func (h *PostHandler) detail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("缺少 slug"))
		return
	}

	res, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/tags
func (h *PostHandler) tags(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ListTags(r.Context())
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/tags/{slug}/posts
func (h *PostHandler) listByTag(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)

	res, err := h.svc.ListByTag(r.Context(), slug, page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/archive
func (h *PostHandler) archive(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Archive(r.Context())
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/archive/{year}/{month}
func (h *PostHandler) listByMonth(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("年份必须是数字"))
		return
	}
	month, err := strconv.Atoi(chi.URLParam(r, "month"))
	if err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("月份必须是数字"))
		return
	}
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)

	res, err := h.svc.ListByMonth(r.Context(), year, month, page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/posts/search?q=...&page=1&page_size=10
func (h *PostHandler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)

	res, err := h.svc.Search(r.Context(), q, page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/users/{publicID} —— 用户公开资料（匿名可读，不含邮箱/权限位）
func (h *PostHandler) authorProfile(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.AuthorProfile(r.Context(), chi.URLParam(r, "publicID"))
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// GET /api/v1/users/{publicID}/posts —— 该用户已发布的文章（匿名可读）
func (h *PostHandler) authorPosts(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)

	res, err := h.svc.ListByAuthor(r.Context(), chi.URLParam(r, "publicID"), page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, res)
}

// queryInt 读取整型查询参数：缺失或非法时用默认值，不报错。
//
// 分页参数不是业务输入，访客改坏了拉回默认值比返回 400 体验更好。
func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
