import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminPostsApi, statusLabel } from '../adminApi'
import { ApiError } from '@/lib/api'
import { formatDate } from '@/lib/format'

type Filter = 'all' | 'draft' | 'published'

const tabs: { key: Filter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'draft', label: '草稿' },
  { key: 'published', label: '已发布' },
]

export function AdminPostListPage() {
  const [filter, setFilter] = useState<Filter>('all')
  const [page, setPage] = useState(1)
  const qc = useQueryClient()
  const navigate = useNavigate()

  const query = useQuery({
    queryKey: ['admin-posts', filter, page],
    queryFn: () => adminPostsApi.list(filter, page, 10),
  })

  const del = useMutation({
    mutationFn: (id: string) => adminPostsApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-posts'] }),
  })

  const toggleStatus = useMutation({
    mutationFn: ({ id, publish }: { id: string; publish: boolean }) =>
      adminPostsApi.setStatus(id, publish),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-posts'] }),
  })

  const data = query.data
  const errMsg =
    query.error instanceof ApiError ? query.error.message : '加载失败'

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-bold">文章管理</h1>
        <Link
          to="/studio/posts/new"
          className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:opacity-90"
        >
          + 写新文章
        </Link>
      </div>

      <div className="mb-4 flex gap-2 border-b border-border">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => {
              setFilter(t.key)
              setPage(1)
            }}
            className={`px-3 py-2 text-sm transition-colors ${
              filter === t.key
                ? 'border-b-2 border-accent font-medium text-accent'
                : 'text-fg-muted hover:text-fg'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {query.isPending && (
        <div className="h-40 animate-pulse rounded border border-border bg-card" />
      )}
      {query.isError && (
        <p className="rounded border border-border bg-card p-4 text-sm text-red-600 dark:text-red-300">
          {errMsg}
        </p>
      )}

      {data && data.items.length === 0 && (
        <p className="rounded border border-border bg-card p-6 text-center text-sm text-fg-muted">
          还没有文章，点击右上角「写新文章」开始。
        </p>
      )}

      {data && data.items.length > 0 && (
        <>
          <ul className="divide-y divide-border overflow-hidden rounded-lg border border-border">
            {data.items.map((p) => (
              <li
                key={p.public_id}
                className="flex items-center gap-4 bg-card px-4 py-3"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => navigate(`/studio/posts/${p.public_id}`)}
                      className="truncate font-medium hover:text-accent"
                    >
                      {p.title || '(无标题)'}
                    </button>
                    <span
                      className={`shrink-0 rounded-full px-2 py-0.5 text-xs ${
                        p.status === 2
                          ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                          : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
                      }`}
                    >
                      {statusLabel(p.status)}
                    </span>
                  </div>
                  <p className="mt-0.5 truncate text-xs text-fg-muted">
                    /{p.slug} · 更新 {formatDate(p.updated_at)}
                    {p.published_at && ` · 发布 ${formatDate(p.published_at)}`}
                  </p>
                </div>

                <div className="flex shrink-0 items-center gap-2 text-sm">
                  <button
                    onClick={() =>
                      toggleStatus.mutate({ id: p.public_id, publish: p.status !== 2 })
                    }
                    className="rounded border border-border px-2.5 py-1 text-fg-muted hover:border-accent hover:text-accent"
                  >
                    {p.status === 2 ? '转草稿' : '发布'}
                  </button>
                  <button
                    onClick={() => navigate(`/studio/posts/${p.public_id}`)}
                    className="rounded border border-border px-2.5 py-1 text-fg-muted hover:border-accent hover:text-accent"
                  >
                    编辑
                  </button>
                  <button
                    onClick={() => {
                      if (confirm(`确定删除《${p.title || p.slug}》？此操作不可恢复。`)) {
                        del.mutate(p.public_id)
                      }
                    }}
                    className="rounded border border-border px-2.5 py-1 text-red-500 hover:border-red-400"
                  >
                    删除
                  </button>
                </div>
              </li>
            ))}
          </ul>

          {data.total > data.page_size && (
            <div className="mt-4 flex justify-center gap-2 text-sm">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="rounded border border-border px-3 py-1 disabled:opacity-40"
              >
                上一页
              </button>
              <span className="px-2 py-1 text-fg-muted">
                {page} / {Math.ceil(data.total / data.page_size)}
              </span>
              <button
                disabled={page * data.page_size >= data.total}
                onClick={() => setPage((p) => p + 1)}
                className="rounded border border-border px-3 py-1 disabled:opacity-40"
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
