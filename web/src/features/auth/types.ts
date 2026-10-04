// 管理员与登录相关的类型。

export interface AdminUser {
  id: number
  public_id: string
  email: string
  display_name: string
  can_upload: boolean
  // 是否拥有发帖/管理权限（管理员开启，普通注册用户默认 false）
  can_publish: boolean
  // 是否为站点管理员（可进入纯管理后台、管理用户与权限）。与 can_publish 解耦。
  is_admin: boolean
  // 公开身份，本人可在「个人资料」页修改。前端不再写死博主信息：
  // 不同账号登录，看到的是各自填的昵称/简介/头像。
  bio: string
  avatar_url: string
  github: string
}

/** 站长公开资料（匿名接口 /site/profile 返回）。刻意不含邮箱。 */
export interface SiteProfile {
  public_id: string
  display_name: string
  bio: string
  avatar_url: string
  github: string
}

export interface LoginResult {
  access_token: string
  user: AdminUser
}

/** 我的视角发布统计（GET /admin/me/stats）：登录态下首页身份区展示本人的数字 */
export interface MyStats {
  post_total: number
  tag_total: number
  month_total: number
}
