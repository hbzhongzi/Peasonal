import { Link, Outlet } from 'react-router-dom'
import { ThemeToggle } from '@/components/ThemeToggle'
import { SearchBox } from '@/components/SearchBox'
import { UserMenu } from '@/components/UserMenu'
import { useAuth } from '@/features/auth/store'
import { site } from '@/config/site'
import { useViewerProfile } from '@/features/site/useSiteProfile'

const navLink =
  'text-sm text-fg-muted transition-colors hover:text-accent'

export function RootLayout() {
  const year = new Date().getFullYear()
  const status = useAuth((s) => s.status)
  // 品牌/页脚跟随当前视角：登录显示登录者的名字（「XX的博客」），匿名显示站长
  const { name } = useViewerProfile()

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-10 border-b border-border bg-bg/85 backdrop-blur">
        <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-6 py-3">
          <Link to="/" className="flex items-center gap-2 font-semibold">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-accent text-sm font-bold text-accent-fg">
              {name.slice(0, 1)}
            </span>
            {name}的博客
          </Link>

          <nav className="flex items-center gap-5">
            <Link to="/" className={navLink}>
              首页
            </Link>
            <Link to="/archive" className={navLink}>
              归档
            </Link>
            <Link to="/tags" className={navLink}>
              标签
            </Link>
            <Link to="/about" className={navLink}>
              关于
            </Link>
            {/* 前台是纯内容站：这里刻意**没有**任何通往 /studio、/admin 的入口，
                工作台与后台由用户直接输 URL 单独访问（用户明确要求，2026-09-29） */}
            {/* 身份模块：头像+昵称是用户菜单的触发器，菜单里只有
                我的主页 / 个人资料 / 退出（不再含「我的博客」「后台」） */}
            {status === 'authed' ? (
              <UserMenu />
            ) : (
              <span className="flex items-center gap-3">
                <Link to="/login" className={navLink}>
                  登录
                </Link>
                <Link
                  to="/register"
                  className="rounded-md bg-accent px-2.5 py-1 text-xs font-medium text-accent-fg hover:opacity-90"
                >
                  注册
                </Link>
              </span>
            )}
            <SearchBox />
            <ThemeToggle />
          </nav>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-6 py-8">
        <Outlet />
      </main>

      <footer className="border-t border-border px-6 py-5">
        <div className="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-2 text-xs text-fg-muted">
          <p>
            © {site.since === year ? site.since : `${site.since}–${year}`} {name}
            <span className="mx-2">·</span>
            Powered by Go + React
          </p>
          {site.icp && (
            <a
              href="https://beian.miit.gov.cn/"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-accent"
            >
              {site.icp}
            </a>
          )}
        </div>
      </footer>
    </div>
  )
}
