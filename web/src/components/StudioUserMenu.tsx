import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'
import { Avatar } from '@/components/Avatar'

const item =
  'block w-full rounded-md px-3 py-2 text-left text-sm text-fg transition-colors hover:bg-bg-subtle hover:text-accent'

// 工作台顶栏的用户菜单：与前台 UserMenu 样式一致。
// 下拉里放「个人资料」和「退出登录」，工作台相关入口仍保留在顶栏导航。
export function StudioUserMenu() {
  const user = useAuth((s) => s.user)
  const logout = useAuth((s) => s.logout)
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const boxRef = useRef<HTMLDivElement>(null)

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
    navigate('/login', { replace: true })
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
