// 认证相关的后端调用。token 注入与 cookie 刷新由 lib/api 统一处理。
import { request } from '@/lib/api'
import type { AdminUser, LoginResult, MyStats, SiteProfile } from './types'

export interface UpdateProfileInput {
  display_name: string
  bio: string
  avatar_url: string
  github: string
}

export interface RegisterInput {
  email: string
  password: string
  display_name?: string
}

export interface UpdatePermsInput {
  can_upload: boolean
  can_publish: boolean
  can_admin: boolean
}

export const authApi = {
  // 密码登录：成功后响应里带 access_token，refresh token 由后端写进 HttpOnly cookie
  login: (email: string, password: string) =>
    request<LoginResult>('POST', '/admin/login', { email, password }),

  // 公开注册：成功后直接签发会话（注册即登录）
  register: (body: RegisterInput) =>
    request<LoginResult>('POST', '/admin/register', body),

  // 用 cookie 里的 refresh token 换新会话（页面刷新时调用）
  refresh: () => request<LoginResult>('POST', '/admin/refresh'),

  logout: () => request<void>('POST', '/admin/logout'),

  // 校验当前 access token 是否仍有效
  me: () => request<AdminUser>('GET', '/admin/me'),

  // 管理员：列出全部用户
  listUsers: () => request<AdminUser[]>('GET', '/admin/users'),

  // 本人：修改自己的公开身份（昵称/简介/头像/GitHub）。后端只认 token 里的 id
  updateMe: (body: UpdateProfileInput) =>
    request<AdminUser>('PATCH', '/admin/me', body),

  // 我的视角发布统计（登录态首页身份区用）：本人的 文章数/标签数/写作月数
  myStats: () => request<MyStats>('GET', '/admin/me/stats'),

  // 管理员：开启/关闭某用户的发帖与上传权限
  updatePerms: (id: string, body: UpdatePermsInput) =>
    request<AdminUser>('PATCH', `/admin/users/${id}`, body),

  // 匿名可读：站长（第一个管理员）的公开资料，前台博主卡片用它渲染
  siteProfile: () => request<SiteProfile>('GET', '/site/profile'),
}
