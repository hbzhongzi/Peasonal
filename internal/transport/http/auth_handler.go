package httpapi

import (
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"

	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/service"
	"personal-blog/internal/transport/http/middleware"
)

// AuthHandler 处理登录、刷新、登出与当前用户信息。
//
// refresh token 走 HttpOnly cookie；access token 通过 JSON 返回，
// 存前端内存（不落 localStorage，防 XSS 窃取）。
type AuthHandler struct {
	auth   *service.AuthService
	auther *middleware.Auther
	log    *slog.Logger
	secure bool // 生产环境为 true：refresh cookie 加 Secure + SameSite=None
}

func NewAuthHandler(authSvc *service.AuthService, auther *middleware.Auther, log *slog.Logger, secure bool) *AuthHandler {
	return &AuthHandler{auth: authSvc, auther: auther, log: log, secure: secure}
}

func (h *AuthHandler) Register(r chi.Router) {
	r.Post("/admin/login", h.login)
	r.Post("/admin/register", h.register)
	r.Post("/admin/refresh", h.refresh)
	r.Post("/admin/logout", h.logout)
}

// RegisterAuthed 注册需要鉴权的路由（/admin/me + 用户管理）。
// 注意：各路由自行声明所需权限级别，不再依赖路由组统一加 RequireAuth。
func (h *AuthHandler) RegisterAuthed(r chi.Router) {
	r.With(h.auther.RequireAuth).Get("/admin/me", h.me)
	// 改自己的资料：任何登录用户都能改，但只能改自己的（userID 取自 token）
	r.With(h.auther.RequireAuth).Patch("/admin/me", h.updateMe)
	// 我的视角发布统计：首页身份区在登录态下展示本人数字（任何登录用户）
	r.With(h.auther.RequireAuth).Get("/admin/me/stats", h.myStats)
	// 用户/权限管理属于「纯管理后台」，只有站点管理员能进（RequireAdmin），
	// 与能否发帖（can_publish，RequirePublish）解耦。
	r.With(h.auther.RequireAdmin).Get("/admin/users", h.listUsers)
	r.With(h.auther.RequireAdmin).Patch("/admin/users/{id}", h.updateUserPerms)
}

// PATCH /api/v1/admin/me  —— 本人修改自己的公开身份（昵称/简介/头像/GitHub）。
//
// 这里刻意不接受 user id：id 只从 token 取。传路径参数就等于开了「改别人资料」的口子。
func (h *AuthHandler) updateMe(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		apierr.WriteError(w, h.log, apierr.Unauthorized("未登录"))
		return
	}
	var req updateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	user, err := h.auth.UpdateMyProfile(r.Context(), p.UserID, service.ProfileInput{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		Github:      req.Github,
	})
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Github      string `json:"github"`
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// POST /api/v1/admin/register  —— 公开注册，成功后直接签发会话（注册即登录）。
func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	res, err := h.auth.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	http.SetCookie(w, middleware.RefreshCookie(res.RefreshToken, h.secure))
	apierr.WriteOK(w, map[string]any{
		"access_token": res.AccessToken,
		"user":         res.User,
	})
}

// GET /api/v1/admin/users  —— 列出全部用户（仅管理员）。
func (h *AuthHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers(r.Context())
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, users)
}

type updatePermsRequest struct {
	CanUpload  bool `json:"can_upload"`
	CanPublish bool `json:"can_publish"`
	CanAdmin   bool `json:"can_admin"`
}

// PATCH /api/v1/admin/users/{id}  —— 开启/关闭某用户的发帖与上传权限，并可升降管理员身份（仅管理员）。
func (h *AuthHandler) updateUserPerms(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updatePermsRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	user, err := h.auth.UpdatePermissions(r.Context(), id, req.CanUpload, req.CanPublish, req.CanAdmin)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, user)
}

// POST /api/v1/admin/login
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		apierr.WriteError(w, h.log, apierr.InvalidArgument("请求体格式错误"))
		return
	}
	res, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	// refresh token 写入 HttpOnly cookie，前端拿不到、XSS 偷不走
	http.SetCookie(w, middleware.RefreshCookie(res.RefreshToken, h.secure))
	apierr.WriteOK(w, map[string]any{
		"access_token": res.AccessToken,
		"user":         res.User,
	})
}

// POST /api/v1/admin/refresh
func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		apierr.WriteError(w, h.log, apierr.Unauthorized("缺少刷新令牌"))
		return
	}
	res, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		// 刷新失败顺手清掉坏 cookie
		http.SetCookie(w, middleware.ClearRefreshCookie(h.secure))
		apierr.WriteError(w, h.log, err)
		return
	}
	http.SetCookie(w, middleware.RefreshCookie(res.RefreshToken, h.secure))
	apierr.WriteOK(w, map[string]any{
		"access_token": res.AccessToken,
		"user":         res.User,
	})
}

// POST /api/v1/admin/logout
func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		_ = h.auth.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, middleware.ClearRefreshCookie(h.secure))
	apierr.WriteNoContent(w)
}

// GET /api/v1/admin/me  —— 必须带 access token（由路由层 RequireAuth 保护）。
func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		apierr.WriteError(w, h.log, apierr.Unauthorized("未登录"))
		return
	}
	user, err := h.auth.Me(r.Context(), p.UserID)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, user)
}

// GET /api/v1/admin/me/stats —— 我的视角发布统计（文章数/标签数/写作月数）。
// 任何登录用户可查，且**只能查自己的**：userID 取自 token，不接受路径参数。
func (h *AuthHandler) myStats(w http.ResponseWriter, r *http.Request) {
	p, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		apierr.WriteError(w, h.log, apierr.Unauthorized("未登录"))
		return
	}
	stats, err := h.auth.MyStats(r.Context(), p.UserID)
	if err != nil {
		apierr.WriteError(w, h.log, err)
		return
	}
	apierr.WriteOK(w, stats)
}
