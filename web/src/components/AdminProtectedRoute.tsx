import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/features/auth/store'

// 后台受保护路由：未登录跳转 /admin/login，并记录来源以便登录后跳回。
export function AdminProtectedRoute({ children }: { children: ReactNode }) {
  const status = useAuth((s) => s.status)
  const location = useLocation()

  if (status === 'loading') {
    return <div className="py-20 text-center text-sm text-fg-muted">正在验证登录状态…</div>
  }
  if (status === 'anon') {
    return <Navigate to="/admin/login" replace state={{ from: location.pathname }} />
  }
  return <>{children}</>
}
