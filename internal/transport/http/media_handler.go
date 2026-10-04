package httpapi

import (
	"io"
	"net/http"
	"strconv"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/service"
	"personal-blog/internal/transport/http/middleware"
)

// MediaHandler 提供媒体库的上传、列表与删除。
//
// 上传接口用 http.MaxBytesReader 在读 body 前就掐断超限请求，
// 服务端不会把超大文件读进内存（对应技术方案 §5.5）。
type MediaHandler struct {
	svc         *service.MediaService
	log         *slog.Logger
	maxUploadMB int64
}

func NewMediaHandler(svc *service.MediaService, log *slog.Logger, maxUploadMB int64) *MediaHandler {
	return &MediaHandler{svc: svc, log: log, maxUploadMB: maxUploadMB}
}

func (h *MediaHandler) Register(r chi.Router, auther *middleware.Auther) {
	// 上传需要 can_upload 权限；列表/删除属于后台管理，要求 can_publish
	r.With(auther.RequireUpload).Post("/admin/media", h.upload)
	r.With(auther.RequirePublish).Get("/admin/media", h.list)
	r.With(auther.RequirePublish).Delete("/admin/media/{id}", h.delete)
}

// POST /api/v1/admin/media  （RequireUpload 由路由层包裹）
func (h *MediaHandler) upload(w http.ResponseWriter, r *http.Request) {
	// 在读 body 前先限流：超过配额的直接 413，且不会把文件读进内存
	maxBytes := h.maxUploadMB << 20 // MB → 字节
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes + (1 << 20)); err != nil {
		apierr.WriteError(w, h.log, apierr.PayloadTooLarge("文件超过大小限制（最大 "+strconv.FormatInt(h.maxUploadMB, 10)+"MB）"))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("缺少上传文件字段 file"))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("读取上传文件失败"))
		return
	}

	item, err := h.svc.Upload(r.Context(), actorOf(r), data)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteCreated(w, item)
}

// GET /api/v1/admin/media?page=&page_size=
func (h *MediaHandler) list(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", service.DefaultPageSize)
	out, err := h.svc.List(r.Context(), actorOf(r), page, pageSize)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}

// DELETE /api/v1/admin/media/{id}
func (h *MediaHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), actorOf(r), chi.URLParam(r, "id")); err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteNoContent(w)
}
