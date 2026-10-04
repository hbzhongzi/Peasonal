// 用户公开资料与其博客页数据。
//
// 为什么独立成 feature：/blog/{publicID} 是「某个用户的博客主页」，
// 与 posts（内容）、auth（登录态）都不是一回事。

import { get } from '@/lib/api'
import type { Page, PostListItem } from '@/features/posts/api'

export interface PublicProfile {
  public_id: string
  display_name: string
  bio: string
  avatar_url: string
  github: string
}

export const usersApi = {
  // 匿名可读：不含邮箱与权限位
  profile: (publicID: string) =>
    get<PublicProfile>(`/users/${encodeURIComponent(publicID)}`),

  // 该用户已发布的文章（分页）
  posts: (publicID: string, page = 1, pageSize = 10) =>
    get<Page<PostListItem>>(`/users/${encodeURIComponent(publicID)}/posts`, {
      page,
      page_size: pageSize,
    }),
}
