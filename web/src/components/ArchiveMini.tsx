import { Link } from 'react-router-dom'
import type { ArchiveMonth } from '@/features/posts/api'

/**
 * 侧边栏的「最近写作」块：取最近 3 个有文章的月份。
 * 完整归档走 /archive，这里只做入口。
 */
export function ArchiveMini({ months }: { months?: ArchiveMonth[] }) {
  if (!months || months.length === 0) {
    return <p className="text-sm text-fg-muted">还没有文章。</p>
  }

  return (
    <ul className="space-y-1.5">
      {months.slice(0, 3).map((m) => (
        <li key={`${m.year}-${m.month}`}>
          <Link
            to={`/archive/${m.year}/${m.month}`}
            className="flex items-center justify-between rounded px-2 py-1 text-sm text-fg-muted transition-colors hover:bg-bg-subtle hover:text-accent"
          >
            <span>{m.label}</span>
            <span className="text-xs tabular-nums">{m.total} 篇</span>
          </Link>
        </li>
      ))}
    </ul>
  )
}
