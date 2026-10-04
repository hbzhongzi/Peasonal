import { Link } from 'react-router-dom'

// 分页走 URL 的 ?page= 而不是组件内 state：
// 这样才能分享链接、用浏览器后退返回上一页。
export function Pagination({
  page,
  pageSize,
  total,
  to,
}: {
  page: number
  pageSize: number
  total: number
  to: (next: number) => string
}) {
  const totalPages = pageSize > 0 ? Math.ceil(total / pageSize) : 0
  if (totalPages <= 1) return null

  const linkClass = 'rounded border border-border px-3 py-1 text-sm hover:border-accent hover:text-accent'
  const disabledClass = 'rounded border border-border px-3 py-1 text-sm text-fg-muted opacity-50'

  return (
    <nav className="mt-8 flex items-center justify-center gap-4">
      {page > 1 ? (
        <Link to={to(page - 1)} className={linkClass} rel="prev">
          上一页
        </Link>
      ) : (
        <span className={disabledClass}>上一页</span>
      )}

      <span className="text-sm text-fg-muted">
        {page} / {totalPages}
      </span>

      {page < totalPages ? (
        <Link to={to(page + 1)} className={linkClass} rel="next">
          下一页
        </Link>
      ) : (
        <span className={disabledClass}>下一页</span>
      )}
    </nav>
  )
}
