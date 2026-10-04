import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { ThemeToggle } from '@/components/ThemeToggle'
import { site } from '@/config/site'

const navItem =
  'rounded-md px-3 py-1.5 text-sm transition-colors hover:bg-bg-subtle'

function navClass({ isActive }: { isActive: boolean }): string {
  return `${navItem} ${isActive ? 'bg-bg-subtle text-accent' : 'text-fg-muted'}`
}

// 纯管理后台：仅站点管理员可进入，用于管理用户与权限。
// 文章/媒体/评论的编辑属于「我的博客」工作台（/studio），不在这里。
export function AdminLayout() {
  const user = useAuth((s) => s.user)
  const canPublish = useAuth((s) => s.canPublish())
  const isAdmin = useAuth((s) => s.isAdmin())
  const logout = useAuth((s) => s.logout)
  const navigate = useNavigate()

  async function onLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  // 已登录但非站点管理员：没有后台管理权限。
  if (!isAdmin) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg-subtle px-4">
        <div className="w-full max-w-sm rounded-xl border border-border bg-card p-8 text-center shadow-sm">
          <p className="text-lg font-semibold">您没有后台管理权限</p>
          <p className="mt-2 text-sm text-fg-muted">
            当前账号（{user?.email}）不是站点管理员，无法进入后台。
            {canPublish && ' 你可在「我的博客」里管理自己的文章（直接输入 /studio 访问）。'}
          </p>
          <div className="mt-5 flex justify-center gap-3">
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
      <header className="sticky top-0 z-10 border-b border-border bg-card">
        <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-6 py-3">
          <Link to="/admin" className="flex items-center gap-2 font-semibold">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-accent text-sm font-bold text-accent-fg">
              {site.name.slice(0, 1)}
            </span>
            后台管理
          </Link>

          <nav className="flex items-center gap-1">
            <NavLink to="/admin" end className={navClass}>
              用户
            </NavLink>
            {/* 评论挂在文章上、留言者是匿名访客，属于全站数据，
                因此管理入口放在纯管理后台而不是「我的博客」。
                后台顶栏不提供任何跳出后台的跳转按钮（用户要求，2026-09-30）。 */}
            <NavLink to="/admin/comments" className={navClass}>
              评论
            </NavLink>
          </nav>

          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-fg-muted sm:inline">
              {user?.display_name}
            </span>
            <ThemeToggle />
            <button
              onClick={onLogout}
              className="rounded-md border border-border px-3 py-1.5 text-sm text-fg-muted transition-colors hover:border-accent hover:text-accent"
            >
              退出
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
