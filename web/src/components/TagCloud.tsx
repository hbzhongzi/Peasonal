import { Link } from 'react-router-dom'
import type { TagWithCount } from '@/features/posts/api'

/**
 * 标签云。
 *
 * 字号随文章数微调（1–3 级），让「写得多的标签」自然突出，
 * 但不做花哨的颜色映射 —— 标签云的职责是导航，不是信息图。
 */
export function TagCloud({ tags, max = 12 }: { tags?: TagWithCount[]; max?: number }) {
  if (!tags || tags.length === 0) {
    return <p className="text-sm text-fg-muted">还没有标签。</p>
  }

  const top = [...tags].sort((a, b) => b.post_count - a.post_count).slice(0, max)
  const peak = Math.max(...top.map((t) => t.post_count), 1)

  const sizeCls = (count: number) => {
    const ratio = count / peak
    if (ratio > 0.66) return 'text-base font-semibold'
    if (ratio > 0.33) return 'text-sm'
    return 'text-xs'
  }

  return (
    <div className="flex flex-wrap gap-x-3 gap-y-2">
      {top.map((t) => (
        <Link
          key={t.slug}
          to={`/tags/${t.slug}`}
          className={`${sizeCls(t.post_count)} text-fg-muted transition-colors hover:text-accent`}
        >
          {t.name}
          <sup className="ml-0.5 text-[10px] text-fg-muted/70">{t.post_count}</sup>
        </Link>
      ))}
    </div>
  )
}
