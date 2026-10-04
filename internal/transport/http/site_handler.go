package httpapi

import (
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/service"
)

// SiteHandler 提供匿名可读的站点级信息。
//
// 目前只有站长公开资料一项：前台首页/关于页要显示「这是谁的博客」，
// 而博主身份不能再写死在前端常量里 —— 那样换个人登录看到的还是上一个人的名字。
type SiteHandler struct {
	auth *service.AuthService
	log  *slog.Logger
}

func NewSiteHandler(authSvc *service.AuthService, log *slog.Logger) *SiteHandler {
	return &SiteHandler{auth: authSvc, log: log}
}

// Register 注册公开路由（不需要任何鉴权）。
func (h *SiteHandler) Register(r chi.Router) {
	r.Get("/site/profile", h.profile)
}

// GET /api/v1/site/profile  —— 站长公开资料（匿名可读，字段最小化）。
func (h *SiteHandler) profile(w http.ResponseWriter, r *http.Request) {
	out, err := h.auth.SiteProfile(r.Context())
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, out)
}
