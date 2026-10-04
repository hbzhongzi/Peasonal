import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { commentsApi } from '../api'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'

// 评论管理（纯管理后台内，RequireAdmin）：按文章查看全部评论（含已隐藏），可隐藏/删除。
//
// 评论挂在文章上、留言者是匿名访客，属于**全站**数据，不是某个作者私有的，
// 所以入口放在 /admin/comments 而不是「我的博客」/studio。
export function AdminCommentsPage() {
  const [slug, setSlug] = useState('')
  const [page, setPage] = useState(1)
  const qc = useQueryClient()

  const query = useQuery({
    queryKey: ['admin-comments', slug, page],
    queryFn: () => commentsApi.listAdmin(slug, page, 30),
    enabled: slug.trim().length > 0,
  })

  const hide = useMutation({
    mutationFn: (id: string) => commentsApi.hide(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-comments'] }),
  })
  const remove = useMutation({
    mutationFn: (id: string) => commentsApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-comments'] }),
  })

  return (
    <div>
      <h1 className="mb-5 text-xl font-bold">评论管理</h1>

      <label className="mb-5 flex flex-col gap-1.5 text-sm">
        <span className="text-fg-muted">输入文章 Slug 查看其评论</span>
        <input
          value={slug}
          onChange={(e) => {
            setSlug(e.target.value)
            setPage(1)
          }}
          placeholder="例如 why-build-my-own-blog"
          className="input max-w-sm"
        />
      </label>

      {slug.trim().length === 0 && (
        <p className="rounded border border-border bg-card p-6 text-center text-sm text-fg-muted">
          输入文章 Slug 后，这里会显示该文章下的全部评论。
        </p>
      )}

      {query.isPending && slug && (
        <div className="h-32 animate-pulse rounded border border-border bg-card" />
      )}
      {query.isError && (
        <p className="rounded border border-border bg-card p-4 text-sm text-red-600 dark:text-red-300">
          {query.error instanceof ApiError ? query.error.message : '加载失败'}
        </p>
      )}

      {query.data && query.data.items.length === 0 && (
        <p className="rounded border border-border bg-card p-6 text-center text-sm text-fg-muted">
          这篇文章还没有评论。
        </p>
      )}

      {query.data && query.data.items.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-lg border border-border">
          {query.data.items.map((c) => (
            <li key={c.public_id} className="flex items-start gap-4 bg-card px-4 py-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 text-sm">
                  <span className="font-medium">{c.author_name}</span>
                  <span className="text-xs text-fg-muted">{formatDateTime(c.created_at)}</span>
                  {c.status !== 1 && (
                    <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
                      已隐藏
                    </span>
                  )}
                </div>
                <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed">{c.content}</p>
              </div>
              <div className="flex shrink-0 flex-col gap-2 text-sm">
                {c.status === 1 ? (
                  <button
                    onClick={() => hide.mutate(c.public_id)}
                    className="rounded border border-border px-2.5 py-1 text-fg-muted hover:border-accent hover:text-accent"
                  >
                    隐藏
                  </button>
                ) : (
                  <button
                    onClick={() => hide.mutate(c.public_id)}
                    className="rounded border border-border px-2.5 py-1 text-fg-muted hover:border-accent hover:text-accent"
                  >
                    恢复
                  </button>
                )}
                <button
                  onClick={() => {
                    if (confirm('确定删除这条评论？')) remove.mutate(c.public_id)
                  }}
                  className="rounded border border-border px-2.5 py-1 text-red-500 hover:border-red-400"
                >
                  删除
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
