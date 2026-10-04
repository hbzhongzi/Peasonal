import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/features/auth/store'
import { authApi } from '@/features/auth/api'
import { Avatar } from '@/components/Avatar'
import { ApiError } from '@/lib/api'
import { siteProfileQuery } from '@/features/site/useSiteProfile'

// 个人资料：本人维护自己的公开身份（昵称/一句话简介/头像/GitHub）。
//
// 这是「不同用户登录看到不一样」的另一半：后端按 token 里的 user_id 读写，
// 页面上没有任何「选一个人来编辑」的入口 —— 你只能改你自己的。
// 权限位（能否发帖/上传/进后台）不在这里改，那是管理员的事。
export function ProfilePage() {
  const user = useAuth((s) => s.user)
  const qc = useQueryClient()

  const [displayName, setDisplayName] = useState('')
  const [bio, setBio] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [github, setGithub] = useState('')
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null)

  // user 是异步 bootstrap 出来的，到位后把表单填一次（之后以用户编辑为准）
  useEffect(() => {
    if (!user) return
    setDisplayName(user.display_name ?? '')
    setBio(user.bio ?? '')
    setAvatarUrl(user.avatar_url ?? '')
    setGithub(user.github ?? '')
  }, [user])

  const save = useMutation({
    mutationFn: () =>
      authApi.updateMe({
        display_name: displayName,
        bio,
        avatar_url: avatarUrl,
        github,
      }),
    onSuccess: (updated) => {
      // 同步内存里的用户态：顶栏昵称/头像要立刻跟着变
      useAuth.setState({ user: updated })
      // 站长资料可能被自己改了（管理员改的就是前台展示的那份）
      qc.invalidateQueries({ queryKey: siteProfileQuery.queryKey })
      setMsg({ ok: true, text: '已保存' })
    },
    onError: (err) => {
      setMsg({
        ok: false,
        text: err instanceof ApiError ? err.message : '保存失败',
      })
    },
  })

  if (!user) return null

  return (
    <div className="max-w-2xl">
      <h1 className="mb-5 text-xl font-bold">个人资料</h1>

      <section className="rounded-lg border border-border bg-card p-6">
        <div className="flex items-center gap-4">
          <Avatar size="lg" name={displayName} avatarUrl={avatarUrl} />
          <div className="min-w-0 text-sm">
            <p className="font-medium">{displayName || '未填昵称'}</p>
            <p className="mt-0.5 text-fg-muted">{user.email}</p>
            <p className="mt-1.5 flex flex-wrap gap-1.5 text-xs text-fg-muted">
              <PermBadge on={user.can_publish} label="可发帖" />
              <PermBadge on={user.can_upload} label="可上传" />
              <PermBadge on={user.is_admin} label="站点管理员" />
            </p>
          </div>
        </div>

        <p className="mt-3 text-xs text-fg-muted">
          邮箱与上面三个权限由管理员在后台管理，这里改不了。
        </p>
      </section>

      <form
        className="mt-5 space-y-4 rounded-lg border border-border bg-card p-6"
        onSubmit={(e) => {
          e.preventDefault()
          setMsg(null)
          save.mutate()
        }}
      >
        <Field label="昵称" hint="前台首页、关于页与页脚显示的名字">
          <input
            className="input w-full"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            maxLength={64}
            required
          />
        </Field>

        <Field label="一句话简介" hint="显示在首页与侧边栏的博主卡片，留空则用默认文案">
          <input
            className="input w-full"
            value={bio}
            onChange={(e) => setBio(e.target.value)}
            maxLength={512}
            placeholder="留空则显示默认简介"
          />
        </Field>

        <Field label="头像地址" hint="填 /uploads/xxx.webp（在媒体库上传后复制），或 https 外链">
          <input
            className="input w-full"
            value={avatarUrl}
            onChange={(e) => setAvatarUrl(e.target.value)}
            placeholder="/uploads/..."
          />
        </Field>

        <Field label="GitHub 主页" hint="填完整链接，前台「找到我」会用它">
          <input
            className="input w-full"
            value={github}
            onChange={(e) => setGithub(e.target.value)}
            placeholder="https://github.com/yourname"
          />
        </Field>

        {msg && (
          <p
            className={
              msg.ok
                ? 'text-sm text-accent'
                : 'text-sm text-red-600 dark:text-red-300'
            }
          >
            {msg.text}
          </p>
        )}

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={save.isPending}
            className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:opacity-90 disabled:opacity-60"
          >
            {save.isPending ? '保存中…' : '保存资料'}
          </button>
          {user.is_admin && (
            <span className="text-xs text-fg-muted">
              你是站点管理员，这份资料同时就是前台展示的博主信息
            </span>
          )}
        </div>
      </form>
    </div>
  )
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <label className="flex flex-col gap-1.5 text-sm">
      <span className="font-medium">{label}</span>
      {children}
      {hint && <span className="text-xs text-fg-muted">{hint}</span>}
    </label>
  )
}

function PermBadge({ on, label }: { on: boolean; label: string }) {
  return (
    <span
      className={
        on
          ? 'rounded-full bg-accent/10 px-2 py-0.5 text-accent'
          : 'rounded-full bg-bg-subtle px-2 py-0.5'
      }
    >
      {label}
      {on ? ' ✓' : ' ✕'}
    </span>
  )
}
