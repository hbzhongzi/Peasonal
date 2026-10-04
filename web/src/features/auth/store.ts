// 认证状态：access token 只存内存（刷新页面即失效），靠 refresh cookie 重新换取。
//
// 设计取舍（见技术方案 §6.4）：
//   - access token 不放 localStorage，避免 XSS 一次性偷走长期凭证。
//   - refresh token 是 HttpOnly cookie，JS 读不到，被偷也用不了（服务端可撤销）。
//   - 任意写操作 401 时由拦截逻辑触发 refresh，UI 无感续期。
import { create } from 'zustand'
import { setAuthToken, setRefreshHandler } from '@/lib/api'
import { authApi } from './api'
import type { AdminUser } from './types'

type AuthStatus = 'loading' | 'authed' | 'anon'

interface AuthState {
  user: AdminUser | null
  status: AuthStatus
  /** 密码登录 */
  login: (email: string, password: string) => Promise<void>
  /** 公开注册：注册成功后直接登录（注册即登录） */
  register: (email: string, password: string, displayName?: string) => Promise<void>
  /** 退出：清 cookie + 清内存 token */
  logout: () => Promise<void>
  /** 应用启动：尝试用 refresh cookie 续期；失败则进入匿名态 */
  bootstrap: () => Promise<void>
  /** 是否拥有发帖/管理权限（默认注册即拥有） */
  canPublish: () => boolean
  /** 是否拥有图片上传权限（由管理员授予，默认关闭） */
  canUpload: () => boolean
  /** 是否为站点管理员（可进入纯管理后台） */
  isAdmin: () => boolean
}

export const useAuth = create<AuthState>((set, get) => ({
  user: null,
  status: 'loading',

  login: async (email, password) => {
    const res = await authApi.login(email, password)
    setAuthToken(res.access_token)
    set({ user: res.user, status: 'authed' })
  },

  register: async (email, password, displayName) => {
    const res = await authApi.register({ email, password, display_name: displayName })
    setAuthToken(res.access_token)
    set({ user: res.user, status: 'authed' })
  },

  logout: async () => {
    try {
      await authApi.logout()
    } catch {
      /* 即使后端失败也本地清态 */
    }
    setAuthToken(null)
    set({ user: null, status: 'anon' })
  },

  bootstrap: async () => {
    try {
      const res = await authApi.refresh()
      setAuthToken(res.access_token)
      set({ user: res.user, status: 'authed' })
    } catch {
      setAuthToken(null)
      set({ user: null, status: 'anon' })
    }
  },

  canPublish: () => get().user?.can_publish ?? false,
  canUpload: () => get().user?.can_upload ?? false,
  isAdmin: () => get().user?.is_admin ?? false,
}))

// 注册 401 自动续期：api 层在任意请求收到 401 时回调这里，用 refresh cookie
// 换新会话并重试原请求。刷新失败则清本地态（cookie 已失效），UI 据此回登录页。
setRefreshHandler(async () => {
  try {
    const res = await authApi.refresh()
    setAuthToken(res.access_token)
    useAuth.setState({ user: res.user })
    return res.access_token
  } catch (e) {
    setAuthToken(null)
    useAuth.setState({ user: null, status: 'anon' })
    throw e
  }
})
