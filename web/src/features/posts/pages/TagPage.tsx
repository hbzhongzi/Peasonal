import { useSearchParams } from 'react-router-dom'
import { Link, useParams } from 'react-router-dom'
import { PostList } from '../components/PostList'
import { Pagination } from '../components/Pagination'
import { usePostsByTag, useTags } from '../hooks'

// GET /tags —— 标签云
export function TagListPage() {
  const { data, isPending, isError, error } = useTags()

  if (isError) {
    return (
      <div className="rounded border border-border bg-card p-6">
        <p className="font-semibold">标签加载失败</p>
        <p className="mt-1 text-sm text-fg-muted">{error.message}</p>
      </div>
    )
  }

  if (isPending) {
    return <div className="h-32 animate-pulse rounded border border-border bg-card" />
  }

  if (data.length === 0) {
    return <p className="py-12 text-center text-fg-muted">还没有标签。</p>
  }

  return (
    <section className="mx-auto max-w-3xl">
      <h1 className="mb-6 text-2xl font-bold">标签</h1>
      <div className="flex flex-wrap gap-3">
        {data.map((t) => (
          <Link
            key={t.slug}
            to={`/tags/${encodeURIComponent(t.slug)}`}
            className="rounded-full border border-border bg-card px-3 py-1 text-sm hover:border-accent hover:text-accent"
          >
            {t.name}
            <span className="ml-1.5 text-xs text-fg-muted">{t.post_count}</span>
          </Link>
        ))}
      </div>
    </section>
  )
}

// GET /tags/:slug —— 某个标签下的文章
export function TagPostsPage() {
  const { slug = '' } = useParams()
  const [params] = useSearchParams()
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1)

  const { data, isPending, isError, error } = usePostsByTag(slug, page)

  if (isError) {
    return (
      <div className="rounded border border-border bg-card p-6">
        <p className="font-semibold">加载失败</p>
        <p className="mt-1 text-sm text-fg-muted">{error.message}</p>
      </div>
    )
  }

  if (isPending) {
    return (
      <div className="space-y-6">
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-24 animate-pulse rounded border border-border bg-card" />
        ))}
      </div>
    )
  }

  return (
    <section className="mx-auto max-w-3xl">
      <h1 className="mb-1 text-2xl font-bold">#{slug}</h1>
      <p className="mb-6 text-sm text-fg-muted">共 {data.total} 篇</p>

      <PostList items={data.items} />
      <Pagination
        page={data.page}
        pageSize={data.page_size}
        total={data.total}
        to={(next) => `/tags/${encodeURIComponent(slug)}?page=${next}`}
      />
    </section>
  )
}
