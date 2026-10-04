// 后台文章管理接口。
import { request } from '@/lib/api'
import type { Page } from './api'

export interface AdminPostListItem {
  public_id: string
  slug: string
  title: string
  summary: string
  cover_url: string
  status: number // 1=草稿 2=已发布 3=归档
  comment_enabled: boolean
  reading_time_min: number
  published_at: string | null
  updated_at: string
}

export interface AdminPostDetail extends AdminPostListItem {
  content_md: string
  tags: string[]
}

export interface PostWriteInput {
  title: string
  slug: string
  summary: string
  content_md: string
  cover_url: string
  comment_enabled: boolean
  tags: string[]
  publish: boolean
}

export const adminPostsApi = {
  list: (status = 'all', page = 1, pageSize = 10) =>
    request<Page<AdminPostListItem>>('GET', '/admin/posts', undefined, {
      status,
      page,
      page_size: pageSize,
    }),

  get: (id: string) => request<AdminPostDetail>('GET', `/admin/posts/${id}`),

  create: (body: PostWriteInput) =>
    request<AdminPostDetail>('POST', '/admin/posts', body),

  update: (id: string, body: PostWriteInput) =>
    request<AdminPostDetail>('PUT', `/admin/posts/${id}`, body),

  remove: (id: string) => request<void>('DELETE', `/admin/posts/${id}`),

  setStatus: (id: string, publish: boolean) =>
    request<void>('PATCH', `/admin/posts/${id}/status`, { publish }),

  setCommentEnabled: (id: string, enabled: boolean) =>
    request<void>('PATCH', `/admin/posts/${id}/comments-enabled`, { enabled }),
}

export function statusLabel(status: number): string {
  switch (status) {
    case 2:
      return '已发布'
    case 3:
      return '已归档'
    default:
      return '草稿'
  }
}
