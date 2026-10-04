// 前端对后端的唯一入口。
//
// 约定：所有请求都经过 get()，统一处理超时、错误码与 {data,error} 解包。
// 组件里不直接写 fetch —— 否则「后端改了响应格式」要改几十处。

import { get, ApiError } from '@/lib/api'

export { ApiError }

export interface Tag {
  slug: string
  name: string
}

export interface TagWithCount extends Tag {
  post_count: number
}

export interface TOCItem {
  id: string
  text: string
  level: number
}

export interface PostAuthor {
  public_id: string
  display_name: string
  avatar_url: string
}

export interface PostListItem {
  public_id: string
  slug: string
  title: string
  summary: string
  cover_url: string
  reading_time_min: number
  published_at: string | null
  tags: Tag[]
  // 作者归属：站上不止一个作者后，「这是谁写的」不再靠猜。
  // 可选是为了容错——作者资料缺失时不至于整块渲染失败。
  author?: PostAuthor
}

export interface Post extends PostListItem {
  content_html: string
  toc: TOCItem[]
  comment_enabled: boolean
}

export interface Page<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface ArchiveMonth {
  year: number
  month: number
  label: string
  total: number
}

export const postsApi = {
  list: (page = 1, pageSize = 10) =>
    get<Page<PostListItem>>('/posts', { page, page_size: pageSize }),

  detail: (slug: string) => get<Post>(`/posts/${encodeURIComponent(slug)}`),

  tags: () => get<TagWithCount[]>('/tags'),

  byTag: (slug: string, page = 1, pageSize = 10) =>
    get<Page<PostListItem>>(`/tags/${encodeURIComponent(slug)}/posts`, { page, page_size: pageSize }),

  archive: () => get<ArchiveMonth[]>('/archive'),

  byMonth: (year: number, month: number, page = 1, pageSize = 10) =>
    get<Page<PostListItem>>(`/archive/${year}/${month}`, { page, page_size: pageSize }),

  search: (q: string, page = 1, pageSize = 10) =>
    get<Page<PostListItem>>('/posts/search', { q, page, page_size: pageSize }),
}
