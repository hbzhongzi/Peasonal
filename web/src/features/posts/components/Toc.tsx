import { useEffect, useState } from 'react'
import type { TOCItem } from '../api'

// 目录 + 滚动高亮。
//
// anchor 与 id 都由后端渲染时生成，前端只负责跳与高亮 —— 不做二次解析 HTML，
// 否则前后端两套 slug 规则迟早对不上。
export function Toc({ items }: { items: TOCItem[] }) {
  const [active, setActive] = useState<string>('')

  useEffect(() => {
    if (items.length === 0) return

    const targets = items
      .map((i) => document.getElementById(i.id))
      .filter((el): el is HTMLElement => el !== null)

    if (targets.length === 0) return

    const visible = new Set<string>()
    const observer = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) visible.add(e.target.id)
          else visible.delete(e.target.id)
        }
        // 取当前可见集合中排在最前的那个，更符合「读到哪儿了」的直觉
        const first = items.find((i) => visible.has(i.id))
        if (first) setActive(first.id)
      },
      // 顶部留出 header 高度，底部大幅收缩，让「当前标题」判定的更准
      { rootMargin: '-80px 0px -70% 0px' },
    )

    targets.forEach((t) => observer.observe(t))
    return () => observer.disconnect()
  }, [items])

  if (items.length === 0) return null

  return (
    <nav className="toc" aria-label="目录">
      <div className="mb-2 text-sm font-semibold text-fg-muted">目录</div>
      <ul className="space-y-1 text-sm">
        {items.map((item) => (
          <li key={item.id} style={{ paddingLeft: (item.level - 1) * 12 }}>
            <a
              href={`#${item.id}`}
              className={
                active === item.id
                  ? 'block border-l-2 border-accent pl-2 text-accent'
                  : 'block border-l-2 border-border pl-2 text-fg-muted hover:text-accent'
              }
              aria-current={active === item.id ? 'true' : undefined}
            >
              {item.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}
