package httpapi

import (
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"personal-blog/internal/domain"
	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/service"
	"personal-blog/internal/transport/http/middleware"
)

// AdminPostHandler 提供文章的后台 CRUD 与状态切换。
type AdminPostHandler struct {
	svc *service.AdminPostService
	log *slog.Logger
}

func NewAdminPostHandler(svc *service.AdminPostService, log *slog.Logger) *AdminPostHandler {
	return &AdminPostHandler{svc: svc, log: log}
}

// Register 的所有路由都要求发帖/管理权限（can_publish）；
// 公开注册的用户默认 can_publish=false，因此无法进入后台管理或发布文章。
func (h *AdminPostHandler) Register(r chi.Router, auther *middleware.Auther) {
	r.With(auther.RequirePublish).Get("/admin/posts", h.list)
	r.With(auther.RequirePublish).Post("/admin/posts", h.create)
	r.With(auther.RequirePublish).Get("/admin/posts/{id}", h.detail)
	r.With(auther.RequirePublish).Put("/admin/posts/{id}", h.update)
	r.With(auther.RequirePublish).Delete("/admin/posts/{id}", h.delete)
	r.With(auther.RequirePublish).Patch("/admin/posts/{id}/status", h.status)
	r.With(auther.RequirePublish).Patch("/admin/posts/{id}/comments-enabled", h.commentsEnabled)
}

type postWriteBody struct {
	Title          string   `json:"title"`
	Slug           string   `json:"slug"`
	Summary        string   `json:"summary"`
	ContentMd      string   `json:"content_md"`
	CoverURL       string   `json:"cover_url"`
	CommentEnabled *bool    `json:"comment_enabled"` // 指针以区分「未传」与「false」
	Tags           []string `json:"tags"`
	Publish        bool     `json:"publish"`
}

func (b postWriteBody) toInput() service.PostUpsertInput {
	enabled := true
	if b.CommentEnabled != nil {
		enabled = *b.CommentEnabled
	}
	return service.PostUpsertInput{
		Title:          b.Title,
		Slug:           b.Slug,
		Summary:        b.Summary,
		ContentMd:      b.ContentMd,
		CoverURL:       b.CoverURL,
		CommentEnabled: enabled,
		Tags:           b.Tags,
		Publish:        b.Publish,
	}
}

// GET /api/v1/admin/posts?page=&page_size=&status=all|draft|published
func (h *AdminPostHandler) list(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)
	statusFilter := statusQueryToInt(r.URL.Query().Get("status"))

	out, err := h.svc.List(r.Context(), actorOf(r), statusFilter, page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}

// POST /api/v1/admin/posts  —— 作者 id 取自 token，不信任请求体
func (h *AdminPostHandler) create(w http.ResponseWriter, r *http.Request) {
	var body postWriteBody
	if err := decodeJSON(r, &body); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	out, err := h.svc.Create(r.Context(), actorOf(r), body.toInput())
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteCreated(w, out)
}

// GET /api/v1/admin/posts/{id}
func (h *AdminPostHandler) detail(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.GetForEdit(r.Context(), actorOf(r), chi.URLParam(r, "id"))
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}

// PUT /api/v1/admin/posts/{id}
func (h *AdminPostHandler) update(w http.ResponseWriter, r *http.Request) {
	var body postWriteBody
	if err := decodeJSON(r, &body); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	out, err := h.svc.Update(r.Context(), actorOf(r), chi.URLParam(r, "id"), body.toInput())
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}

// DELETE /api/v1/admin/posts/{id}
func (h *AdminPostHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), actorOf(r), chi.URLParam(r, "id")); err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteNoContent(w)
}

// PATCH /api/v1/admin/posts/{id}/status  body: {"publish": true|false}
func (h *AdminPostHandler) status(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Publish bool `json:"publish"`
	}
	if err := decodeJSON(r, &body); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	if err := h.svc.SetStatus(r.Context(), actorOf(r), chi.URLParam(r, "id"), body.Publish); err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteNoContent(w)
}

// PATCH /api/v1/admin/posts/{id}/comments-enabled  body: {"enabled": true|false}
func (h *AdminPostHandler) commentsEnabled(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	if err := h.svc.SetCommentEnabled(r.Context(), actorOf(r), chi.URLParam(r, "id"), body.Enabled); err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteNoContent(w)
}

// statusQueryToInt 把语义化状态参数映射到数据库状态值（0 表示不过滤）。
func statusQueryToInt(s string) int8 {
	switch s {
	case "draft":
		return int8(domain.StatusDraft)
	case "published":
		return int8(domain.StatusPublished)
	default:
		return 0
	}
}
