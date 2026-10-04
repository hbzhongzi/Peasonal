import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { ApiError } from '@/lib/api'
import { site } from '@/config/site'

export function LoginPage() {
  const login = useAuth((s) => s.login)
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
      // 登录落地：回来源页（从 /studio、/admin 被踢出来再回去的场景）；
      // 没有来源就回博客首页。**不按权限自动分流进后台**——
      // 前台不提供任何进入后台的路径，工作台/后台由用户直接输 URL 单独访问。
      navigate(redirectFrom ?? '/', { replace: true })
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
          <h1 className="text-xl font-bold">登录</h1>
          <p className="mt-1 text-sm text-fg-muted">{site.name}的博客</p>
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
              placeholder="you@example.com"
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
            {busy ? '登录中…' : '登录'}
          </button>
        </form>

        <p className="mt-5 text-center text-xs text-fg-muted">
          还没有账号？{' '}
          <Link to="/register" className="text-accent hover:underline">
            去注册
          </Link>
          <span className="mx-2">·</span>
          <Link to="/" className="hover:text-accent">
            返回首页
          </Link>
        </p>
      </div>
    </div>
  )
}
