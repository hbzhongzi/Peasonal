package service

import (
	"personal-blog/internal/pkg/apierr"
)

// ErrNotOwner 是归属校验失败的统一错误：非管理员碰了别人的数据。
//
// 为什么要单独定义：文章/媒体/评论都要做同样的判断，分散写会写出三种不同文案，
// 前端也不好 switch。统一成一个变量后，改文案只用改一处。
var ErrNotOwner = apierr.Forbidden("只能操作自己创建的内容")

// Actor 描述「谁在操作」，是数据归属隔离的唯一入口。
//
// 规则很简单：
//   - 每个人（包括站点管理员）在「我的博客」工作台（/studio）只能看到/改自己创建的数据；
//   - 站点管理员通过纯管理后台（/admin）管理用户、权限、评论，不在这里越权编辑他人文章。
//
// 为什么不做成更细的权限（如「可编辑他人文章」）：这是个人博客，
// 作者之间本就该互不干涉，多一个维度就多一堆组合要测。
type Actor struct {
	UserID  uint64
	IsAdmin bool
}

// AuthorFilter 返回传给 SQL 的作者过滤值。
//
// 始终返回当前用户 ID，不允许管理员在 /studio 看全站文章/媒体。
// 仓储层的查询仍统一写成 `WHERE (? = 0 OR author_id = ?)`，
// 这样既能让 sqlc 静态分析（不需要在 Go 里拼 SQL），又能少写一条查询；
// 返回非零值时只有 author_id == UserID 的行会命中。
func (a Actor) AuthorFilter() uint64 {
	return a.UserID
}

// CanTouch 判断能否操作 ownerID 名下的资源。
//
// 注意 ownerID == 0 的历史遗留数据（迁移前没有归属列）只有管理员能动：
// 普通作者的 UserID 从 1 起，永远不等于 0，因此天然被挡在外面。
func (a Actor) CanTouch(ownerID uint64) bool {
	return a.IsAdmin || ownerID == a.UserID
}
