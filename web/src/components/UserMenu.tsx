import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { Avatar } from '@/components/Avatar'

const item =
  'block w-full rounded-md px-3 py-2 text-left text-sm text-fg transition-colors hover:bg-bg-subtle hover:text-accent'

// 顶栏用户菜单：登录后「我是谁、我能去哪」都收在这里。
// 条目：我的主页（回到网站首页 /）、个人资料（/studio/profile，本人身份维护）。
// **没有**「我的博客」「后台」——前台不提供任何进入工作台/后台的入口，
// 两者由用户直接输 URL 单独访问。
export function UserMenu() {
  const user = useAuth((s) => s.user)
  const logout = useAuth((s) => s.logout)
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)

  // 点外面 / 按 Esc 关闭
  useEffect(() => {
    if (!open) return
    function onDown(e: MouseEvent) {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  if (!user) return null

  async function onLogout() {
    setOpen(false)
    await logout()
    navigate('/', { replace: true })
  }

  return (
    <div ref={boxRef} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 rounded-full border border-border py-0.5 pr-2.5 pl-0.5 transition-colors hover:border-accent"
        aria-haspopup="menu"
        aria-expanded={open}
        title="账号菜单"
      >
        <Avatar size="sm" name={user.display_name} avatarUrl={user.avatar_url} />
        <span className="max-w-24 truncate text-sm">{user.display_name || '我'}</span>
        <span aria-hidden className="text-[10px] text-fg-muted">
          ▾
        </span>
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full z-20 mt-2 w-52 rounded-lg border border-border bg-card p-1.5 shadow-lg"
        >
          <div className="border-b border-border px-3 py-2">
            <p className="truncate text-sm font-medium">{user.display_name || '未填昵称'}</p>
            <p className="mt-0.5 truncate text-xs text-fg-muted">{user.email}</p>
          </div>
          <div className="pt-1.5">
            {/* 我的主页 = 回到网站首页 */}
            <Link to="/" className={item} onClick={() => setOpen(false)}>
              我的主页
            </Link>
            <Link to="/studio/profile" className={item} onClick={() => setOpen(false)}>
              个人资料
            </Link>
            <div className="my-1.5 border-t border-border" />
            <button onClick={onLogout} className={item + ' text-fg-muted'}>
              退出登录
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
