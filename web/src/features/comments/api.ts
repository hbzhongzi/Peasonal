// 评论接口：公开读取/提交 + 后台管理（隐藏/删除）。
import { get, request } from '@/lib/api'
import type { Page } from '@/features/posts/api'

export interface Comment {
  public_id: string
  author_name: string
  content: string
  created_at: string
}

export interface AdminComment extends Comment {
  status: number // 1=可见 2=已隐藏
}

export interface CommentSubmit {
  author_name: string
  author_email?: string
  content: string
  hp?: string // 蜜罐：留空
}

export const commentsApi = {
  // 公开：某篇文章的可见评论（分页，按时间正序）
  listPublic: (slug: string, page = 1, pageSize = 20) =>
    get<Page<Comment>>(`/posts/${encodeURIComponent(slug)}/comments`, {
      page,
      page_size: pageSize,
    }),

  // 公开：某篇文章的可见评论总数（评论图标角标用，不拉内容）
  count: (slug: string) =>
    get<{ total: number }>(`/posts/${encodeURIComponent(slug)}/comments/count`),

  // 公开：提交评论（无审核，提交即通过；蜜罐/限流在服务端拦截）
  submit: (slug: string, body: CommentSubmit) =>
    request<Comment>('POST', `/posts/${encodeURIComponent(slug)}/comments`, body),

  // 后台：某篇文章的全部评论（含已隐藏）
  listAdmin: (slug: string, page = 1, pageSize = 30) =>
    request<Page<AdminComment>>('GET', '/admin/comments', undefined, {
      slug,
      page,
      page_size: pageSize,
    }),

  hide: (id: string) => request<void>('PATCH', `/admin/comments/${id}/hide`),

  remove: (id: string) => request<void>('DELETE', `/admin/comments/${id}`),
}
