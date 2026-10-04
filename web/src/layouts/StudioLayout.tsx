import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Avatar } from '@/components/Avatar'
import { StudioUserMenu } from '@/components/StudioUserMenu'

const navItem =
  'rounded-md px-3 py-1.5 text-sm transition-colors hover:bg-bg-subtle'

function navClass({ isActive }: { isActive: boolean }): string {
  return `${navItem} ${isActive ? 'bg-bg-subtle text-accent' : 'text-fg-muted'}`
}

// 「我的博客」：个人博客编辑管理界面。拥有发帖权限(can_publish)的用户可进入，
// 在此写文章、管媒体、维护自己的个人资料。它与「后台管理」(/admin) 是两套独立的东西——
// 普通作者能进这里，但进不了后台；**这里也没有任何通往后台的链接**，后台只认 URL 直访。
//
// 数据按人隔离：文章列表、媒体库都只返回本人创建的（管理员除外），
// 判断在后端做（service.Actor），前端不做任何「假装看不见」的裁剪。
export function StudioLayout() {
  const user = useAuth((s) => s.user)
  const canPublish = useAuth((s) => s.canPublish())
  const logout = useAuth((s) => s.logout)
  const navigate = useNavigate()
  const { pathname } = useLocation()

  async function onLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  // 个人资料只属于「本人」，与发帖权限无关 —— 普通用户也必须能改自己的昵称/头像。
  // 权限墙只拦编辑功能本身（文章/媒体），不拦个人资料。
  const isProfileRoute = pathname.startsWith('/studio/profile')

  // 已登录但无发帖权限的普通用户：没有博客编辑界面可进（个人资料除外）。
  if (!canPublish && !isProfileRoute) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg-subtle px-4">
        <div className="w-full max-w-sm rounded-xl border border-border bg-card p-8 text-center shadow-sm">
          <p className="text-lg font-semibold">您还没有发布权限</p>
          <p className="mt-2 text-sm text-fg-muted">
            当前账号（{user?.email}）只能浏览与评论，无法发布文章。
            如需发布，请联系管理员开启权限。
          </p>
          <div className="mt-5 flex justify-center gap-3">
            <Link to="/" className="rounded-md border border-border px-4 py-2 text-sm hover:border-accent hover:text-accent">
              返回博客首页
            </Link>
            <button
              onClick={onLogout}
              className="rounded-md border border-border px-4 py-2 text-sm text-fg-muted hover:border-accent hover:text-accent"
            >
              退出登录
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen flex-col">
      {/* 上栏样式与前台 RootLayout 保持一致：毛玻璃背景，不再用实色 bg-card */}
      <header className="sticky top-0 z-10 border-b border-border bg-bg/85 backdrop-blur">
        <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-6 py-3">
          {/* 这里显示的是**当前登录者**自己的身份，不是站长资料：
              换个人登录，这里就该换个人的头像与昵称 */}
          <Link to="/studio/profile" className="flex items-center gap-2 font-semibold">
            <Avatar size="sm" name={user?.display_name} avatarUrl={user?.avatar_url} />
            {canPublish ? '我的博客' : '账号'}
          </Link>

          <nav className="flex items-center gap-1">
            <NavLink to="/" end className={navClass}>
              首页
            </NavLink>
            {/* 文章属于编辑功能，只给有发帖权限的人看 */}
            {canPublish && (
              <NavLink to="/studio" end className={navClass}>
                文章
              </NavLink>
            )}
            {/* 刻意**没有**「后台」「评论」入口；但保留「首页」作为返回前台的出口。
                后台只由用户直接输 URL 访问（用户明确要求，2026-09-29）。
                「个人资料」已收到右上角用户菜单下拉中，与前台 UserMenu 保持一致。 */}
          </nav>

          <div className="flex items-center gap-3">
            <ThemeToggle />
            <StudioUserMenu />
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
