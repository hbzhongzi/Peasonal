import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { ApiError } from '@/lib/api'
import { site } from '@/config/site'

// 管理员专属登录页。
//
// 与普通登录页共用 `/api/v1/admin/login`，但登录后必须校验 `is_admin`：
// 非管理员账号即使密码正确也无法进入后台，直接提示「只有站点管理员能登录后台」。
// 这样前台用户不会误入 /admin，后台管理员也有独立的入口与视觉。
export function AdminLoginPage() {
  const login = useAuth((s) => s.login)
  const logout = useAuth((s) => s.logout)
  const isAdmin = useAuth((s) => s.isAdmin)
  const navigate = useNavigate()
  const location = useLocation()
  const redirectFrom = (location.state as { from?: string } | null)?.from

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await login(email.trim(), password)
      if (!isAdmin()) {
        // 不是管理员：清掉刚拿到的会话，不让其停留在后台登录态
        await logout()
        setError('只有站点管理员能登录后台')
        return
      }
      // 管理员登录后回来源页（从 /admin 被踢出来的情况）或进后台首页
      navigate(redirectFrom ?? '/admin', { replace: true })
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : '登录失败，请重试'
      setError(msg)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-subtle px-4">
      <div className="w-full max-w-sm rounded-xl border border-border bg-card p-8 shadow-sm">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-3 flex h-11 w-11 items-center justify-center rounded-lg bg-accent text-lg font-bold text-accent-fg">
            {site.name.slice(0, 1)}
          </div>
          <h1 className="text-xl font-bold">后台管理登录</h1>
          <p className="mt-1 text-sm text-fg-muted">仅站点管理员可进入</p>
        </div>

        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5 text-sm">
            <span className="text-fg-muted">邮箱</span>
            <input
              type="email"
              required
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="rounded-md border border-border bg-bg px-3 py-2 text-fg outline-none focus:border-accent"
              placeholder="admin@example.com"
            />
          </label>

          <label className="flex flex-col gap-1.5 text-sm">
            <span className="text-fg-muted">密码</span>
            <input
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="rounded-md border border-border bg-bg px-3 py-2 text-fg outline-none focus:border-accent"
              placeholder="••••••••"
            />
          </label>

          {error && (
            <p className="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={busy}
            className="mt-1 rounded-md bg-accent px-4 py-2 font-medium text-accent-fg transition-opacity hover:opacity-90 disabled:opacity-60"
          >
            {busy ? '登录中…' : '登录后台'}
          </button>
        </form>

        <p className="mt-5 text-center text-xs text-fg-muted">
          <Link to="/" className="hover:text-accent">
            返回博客首页
          </Link>
        </p>
      </div>
    </div>
  )
}
