import { useSearchParams } from 'react-router-dom'
import { Link, useParams } from 'react-router-dom'
import { PostList } from '../components/PostList'
import { Pagination } from '../components/Pagination'
import { useArchive, usePostsByMonth } from '../hooks'

// GET /archive —— 按月的归档索引
export function ArchivePage() {
  const { data, isPending, isError, error } = useArchive()

  if (isError) {
    return (
      <div className="rounded border border-border bg-card p-6">
        <p className="font-semibold">归档加载失败</p>
        <p className="mt-1 text-sm text-fg-muted">{error.message}</p>
      </div>
    )
  }

  if (isPending) {
    return <div className="h-64 animate-pulse rounded border border-border bg-card" />
  }

  if (data.length === 0) {
    return <p className="py-12 text-center text-fg-muted">还没有归档内容。</p>
  }

  return (
    <section className="mx-auto max-w-3xl">
      <h1 className="mb-6 text-2xl font-bold">归档</h1>
      <ul className="space-y-2">
        {data.map((m) => (
          <li key={`${m.year}-${m.month}`} className="flex items-center justify-between">
            <Link
              to={`/archive/${m.year}/${String(m.month).padStart(2, '0')}`}
              className="hover:text-accent"
            >
              {m.label}
            </Link>
            <span className="text-sm text-fg-muted">{m.total} 篇</span>
          </li>
        ))}
      </ul>
    </section>
  )
}

// GET /archive/:year/:month —— 某个月的文章
export function ArchiveMonthPage() {
  const { year = '', month = '' } = useParams()
  const [params] = useSearchParams()
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1)

  const y = Number(year)
  const m = Number(month)
  const valid = Number.isInteger(y) && Number.isInteger(m) && m >= 1 && m <= 12

  const { data, isPending, isError, error } = usePostsByMonth(y, m, page)

  if (!valid) {
    return <p className="py-12 text-center text-fg-muted">月份不合法。</p>
  }

  if (isError) {
    return (
      <div className="rounded border border-border bg-card p-6">
        <p className="font-semibold">加载失败</p>
        <p className="mt-1 text-sm text-fg-muted">{error.message}</p>
      </div>
    )
  }

  if (isPending) {
    return <div className="h-64 animate-pulse rounded border border-border bg-card" />
  }

  return (
    <section className="mx-auto max-w-3xl">
      <h1 className="mb-1 text-2xl font-bold">
        {y} 年 {m} 月
      </h1>
      <p className="mb-6 text-sm text-fg-muted">共 {data.total} 篇</p>

      <PostList items={data.items} />
      <Pagination
        page={data.page}
        pageSize={data.page_size}
        total={data.total}
        to={(next) => `/archive/${y}/${month}?page=${next}`}
      />
    </section>
  )
}
