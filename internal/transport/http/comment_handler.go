package httpapi

import (
	"net"
	"net/http"
	"strings"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"personal-blog/internal/domain"
	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/service"
	"personal-blog/internal/transport/http/middleware"
)

// CommentHandler 提供评论的公开提交/列表，以及后台的隐藏/删除。
type CommentHandler struct {
	svc *service.CommentService
	log *slog.Logger
}

func NewCommentHandler(svc *service.CommentService, log *slog.Logger) *CommentHandler {
	return &CommentHandler{svc: svc, log: log}
}

func (h *CommentHandler) RegisterPublic(r chi.Router) {
	r.Get("/posts/{slug}/comments", h.publicList)
	r.Get("/posts/{slug}/comments/count", h.publicCount)
	r.Post("/posts/{slug}/comments", h.publicCreate)
}

// RegisterAdmin 注册评论管理路由。
//
// 评论是**全站**的（评论挂在文章上、留言者是匿名访客），不属于「我的博客」的私人数据，
// 所以管理入口放在纯管理后台、用 RequireAdmin 守卫 —— 作者只能管自己文章下的评论，
// 那是通过前台文章页进入的（将来做），不是这里的全站列表。
func (h *CommentHandler) RegisterAdmin(r chi.Router, auther *middleware.Auther) {
	r.With(auther.RequireAdmin).Get("/admin/comments", h.adminList)
	r.With(auther.RequireAdmin).Patch("/admin/comments/{id}/hide", h.adminHide)
	r.With(auther.RequireAdmin).Delete("/admin/comments/{id}", h.adminDelete)
}

type commentWriteBody struct {
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	Content     string `json:"content"`
	Honeypot    string `json:"hp"` // 蜜罐字段，正常表单不会提交
}

// GET /api/v1/posts/{slug}/comments?page=1&page_size=20
// 按时间正序分页返回公开评论；配合前端滑动加载更多。
func (h *CommentHandler) publicList(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	out, err := h.svc.ListPublicPaginated(r.Context(), chi.URLParam(r, "slug"), page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}

// GET /api/v1/posts/{slug}/comments/count
// 仅返回公开评论总数，供文章页的评论图标展示角标，避免为计数拉取全部内容。
func (h *CommentHandler) publicCount(w http.ResponseWriter, r *http.Request) {
	total, err := h.svc.CountPublic(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, map[string]int64{"total": total})
}

// POST /api/v1/posts/{slug}/comments
func (h *CommentHandler) publicCreate(w http.ResponseWriter, r *http.Request) {
	var body commentWriteBody
	if err := decodeJSON(r, &body); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	in := domain.CommentInput{
		AuthorName:  body.AuthorName,
		AuthorEmail: body.AuthorEmail,
		Content:     body.Content,
		Honeypot:    body.Honeypot,
	}
	out, err := h.svc.Create(r.Context(), chi.URLParam(r, "slug"), clientIP(r), in)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteCreated(w, out)
}

// GET /api/v1/admin/comments?slug=xxx  —— 后台按文章查评论（含已隐藏）
func (h *CommentHandler) adminList(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("缺少 slug 参数"))
		return
	}
	// 先解析 slug 到 post id
	post, err := h.svc.PostIDBySlug(r.Context(), slug)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)
	out, err := h.svc.ListByPostAdmin(r.Context(), post, page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}

// PATCH /api/v1/admin/comments/{id}/hide
func (h *CommentHandler) adminHide(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Hide(r.Context(), chi.URLParam(r, "id")); err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteNoContent(w)
}

// DELETE /api/v1/admin/comments/{id}
func (h *CommentHandler) adminDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteNoContent(w)
}

// clientIP 尽力取出真实客户端 IP（本地开发直接读 RemoteAddr）。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个（最原始的客户端）
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
