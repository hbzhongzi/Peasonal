package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/pkg/auth"
)

const tokenIssuer = "personal-blog"

// Principal 是从 access token 载荷取出的当前登录用户。
//
// 之所以塞进上下文而不是每次查库：can_upload 等字段已经写进 JWT，
// 鉴权中间件解析后直接可用，避免每个受保护请求都打一次库。
type Principal struct {
	UserID      uint64
	Email       string
	CanUpload   bool
	CanPublish  bool
	IsAdmin     bool
}

type ctxKey int

const principalKey ctxKey = iota

// Auther 校验 access token 并提供鉴权中间件。
type Auther struct {
	mgr *auth.Manager
}

// NewAuther 用同一个 JWT 密钥与签发方构造校验器。
//
// accessTTL 仅用于构造 Manager；解析 token 时只看 token 自身的 exp 声明，
// 与这里的值无关，传 15m 仅为与签发侧一致。
func NewAuther(secret string) *Auther {
	return &Auther{mgr: auth.NewManager(secret, tokenIssuer, 15*time.Minute)}
}

// PrincipalFrom 取出上下文里的用户信息。
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// RequireAuth 要求合法 access token，否则返回 401。
func (a *Auther) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := a.parse(r)
		if err != nil {
			apierr.WriteError(w, nil, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireUpload 在 RequireAuth 基础上还要求 can_upload 权限。
func (a *Auther) RequireUpload(next http.Handler) http.Handler {
	return a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		if !p.CanUpload {
			apierr.WriteError(w, nil, apierr.Forbidden("当前账号没有上传权限"))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequirePublish 要求发帖/管理权限（can_publish），否则 403。
//
// 公开注册的用户默认 can_publish=false，因此无法发布文章或进入「我的博客」编辑界面，
// 只能评论/上传；发帖权限由管理员在用户管理页开启（见开发规划 M2 补充）。
func (a *Auther) RequirePublish(next http.Handler) http.Handler {
	return a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		if !p.CanPublish {
			apierr.WriteError(w, nil, apierr.Forbidden("当前账号没有发布/管理权限"))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequireAdmin 要求站点管理员权限（is_admin），否则 403。
//
// 这是「纯管理后台」的守卫：只有 is_admin=1 的账号能进入用户/权限管理界面，
// 与能否发帖（can_publish）解耦——普通作者进不了后台。
func (a *Auther) RequireAdmin(next http.Handler) http.Handler {
	return a.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		if !p.IsAdmin {
			apierr.WriteError(w, nil, apierr.Forbidden("只有站点管理员才能访问后台管理"))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (a *Auther) parse(r *http.Request) (Principal, error) {
	raw := r.Header.Get("Authorization")
	if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
		return Principal{}, apierr.Unauthorized("缺少访问令牌")
	}
	token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	claims, err := a.mgr.ParseAccessToken(token)
	if err != nil {
		return Principal{}, apierr.Unauthorized("访问令牌无效或已过期")
	}
	uid, err := claims.UserID()
	if err != nil {
		return Principal{}, apierr.Unauthorized("访问令牌无效")
	}
	return Principal{UserID: uid, Email: claims.Email, CanUpload: claims.CanUpload, CanPublish: claims.CanPublish, IsAdmin: claims.IsAdmin}, nil
}

// RefreshCookie 构造 refresh token 的 HttpOnly cookie。
//
// 本地开发（production=false）用 SameSite=Lax，因为前端（:5173）与后端（:8080）
// 同站；生产环境跨域部署时改用 SameSite=None + Secure。
func RefreshCookie(value string, production bool) *http.Cookie {
	sameSite := http.SameSiteLaxMode
	if production {
		sameSite = http.SameSiteNoneMode
	}
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   production,
		SameSite: sameSite,
		MaxAge:   60 * 60 * 24 * 7, // 7 天
	}
}

// ClearRefreshCookie 让客户端清除 refresh cookie。
func ClearRefreshCookie(production bool) *http.Cookie {
	c := RefreshCookie("", production)
	c.MaxAge = -1
	return c
}
