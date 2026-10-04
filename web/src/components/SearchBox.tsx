import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

// 顶部搜索框：回车跳转到 /search?q=…（中文按 ngram 全文索引检索）。
export function SearchBox() {
  const [q, setQ] = useState('')
  const navigate = useNavigate()

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const term = q.trim()
    if (term) navigate(`/search?q=${encodeURIComponent(term)}`)
  }

  return (
    <form onSubmit={onSubmit} className="relative hidden md:block">
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="搜索文章…"
        className="w-44 rounded-full border border-border bg-bg-subtle py-1.5 pl-9 pr-3 text-sm outline-none transition-all focus:w-56 focus:border-accent"
      />
      <svg
        className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-fg-muted"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <circle cx="11" cy="11" r="7" />
        <path d="m20 20-3.5-3.5" />
      </svg>
    </form>
  )
}
