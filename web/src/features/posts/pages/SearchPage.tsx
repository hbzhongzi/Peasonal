import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ApiError } from '@/lib/api'
import { postsApi } from '../api'
import { PostList } from '../components/PostList'

// 搜索结果页：读取 ?q= 参数，调用后端 ngram 全文检索。
export function SearchPage() {
  const [params] = useSearchParams()
  const q = params.get('q') ?? ''

  const query = useQuery({
    queryKey: ['search', q],
    queryFn: () => postsApi.search(q, 1, 20),
    enabled: q.length > 0,
  })

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-1 text-2xl font-bold">搜索</h1>
      <p className="mb-6 text-sm text-fg-muted">
        关键词：<span className="text-fg">{q}</span>
      </p>

      {query.isPending && (
        <div className="h-40 animate-pulse rounded border border-border bg-card" />
      )}
      {query.isError && (
        <p className="rounded border border-border bg-card p-4 text-sm text-red-600 dark:text-red-300">
          {query.error instanceof ApiError ? query.error.message : '搜索失败'}
        </p>
      )}

      {query.data && query.data.total === 0 && (
        <p className="rounded border border-border bg-card p-6 text-center text-sm text-fg-muted">
          没有找到与「{q}」相关的文章。
        </p>
      )}

      {query.data && query.data.total > 0 && (
        <>
          <p className="mb-4 text-xs text-fg-muted">共 {query.data.total} 条结果</p>
          <PostList items={query.data.items} />
          <p className="mt-6 text-center">
            <Link to="/" className="text-sm text-accent hover:underline">
              ← 返回首页
            </Link>
          </p>
        </>
      )}
    </div>
  )
}
