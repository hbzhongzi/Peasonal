import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { ApiError } from '@/lib/api'
import { site } from '@/config/site'

export function RegisterPage() {
  const register = useAuth((s) => s.register)
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await register(email.trim(), password, displayName.trim() || undefined)
      // 注册即登录。新用户默认只有评论/上传权限，没有发帖与管理权限，
      // 因此回博客首页（而不是 /admin 或 /studio —— 那两处都会显示"无权限"）。
      navigate('/', { replace: true })
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : '注册失败，请重试'
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
          <h1 className="text-xl font-bold">注册账号</h1>
          <p className="mt-1 text-sm text-fg-muted">{site.name}的博客</p>
        </div>

        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5 text-sm">
            <span className="text-fg-muted">邮箱</span>
            <input
              type="email"
              required
              autoComplete="username"
              pattern="[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}"
              title="请输入有效邮箱，如 you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="rounded-md border border-border bg-bg px-3 py-2 text-fg outline-none focus:border-accent"
              placeholder="you@example.com"
            />
          </label>

          <label className="flex flex-col gap-1.5 text-sm">
            <span className="text-fg-muted">密码（至少 8 位）</span>
            <input
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="rounded-md border border-border bg-bg px-3 py-2 text-fg outline-none focus:border-accent"
              placeholder="••••••••"
            />
          </label>

          <label className="flex flex-col gap-1.5 text-sm">
            <span className="text-fg-muted">昵称（可选）</span>
            <input
              type="text"
              autoComplete="nickname"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="rounded-md border border-border bg-bg px-3 py-2 text-fg outline-none focus:border-accent"
              placeholder="留空则用邮箱前缀"
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
            {busy ? '注册中…' : '注册并登录'}
          </button>
        </form>

        <p className="mt-5 text-center text-xs text-fg-muted">
          已有账号？{' '}
          <Link to="/login" className="text-accent hover:underline">
            去登录
          </Link>
        </p>
      </div>
    </div>
  )
}
