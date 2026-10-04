package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"personal-blog/internal/service"
	"personal-blog/internal/transport/http/middleware"
)

// maxJSONBody 限制 JSON 请求体的大小，防止超大 body 撑爆内存。
const maxJSONBody = 1 << 20 // 1MB

// decodeJSON 读取并解析 JSON 请求体，带大小上限。
func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBody))
	return dec.Decode(dst)
}

// actorOf 从请求上下文取出操作者，交给 service 做归属校验。
//
// 只有被 RequireAuth/RequirePublish/RequireAdmin 包裹的路由才会走到这里，
// 否则 PrincipalFrom 返回 false —— 那种情况下 UserID 为 0，
// 任何归属校验都会失败（没有资源属于 id=0），不会「漏权限」。
func actorOf(r *http.Request) service.Actor {
	p, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		return service.Actor{}
	}
	return service.Actor{UserID: p.UserID, IsAdmin: p.IsAdmin}
}
