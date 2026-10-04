import { useEffect, useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { commentsApi, type Comment } from '../api'
import { useCommentCount, useComments } from '../hooks'
import { useAuth } from '@/features/auth/store'
import { ApiError } from '@/lib/api'
import { formatDate } from '@/lib/format'

// 评论图标（对话气泡），用当前文字颜色描边，随主题自适应。
function CommentIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M21 11.5a8.38 8.38 0 0 1-8.5 8.5 8.5 8.5 0 0 1-3.9-.9L3 21l1.9-5.6A8.5 8.5 0 1 1 21 11.5z" />
    </svg>
  )
}

/**
 * 文章页评论区。
 *
 * 默认折叠：只显示一个「评论图标 + 数量」的按钮，点开才加载并展示历史评论。
 * 历史评论用滚动分页（IntersectionObserver 触底拉下一页），不一次全量返回。
 */
export function CommentSection({ slug, enabled }: { slug: string; enabled: boolean }) {
  const [open, setOpen] = useState(false)
  const { data } = useCommentCount(slug, enabled)
  const total = data?.total ?? 0

  if (!enabled) {
    return (
      <section className="mt-12 border-t border-border pt-6 text-sm text-fg-muted">
        本文关闭了评论。
      </section>
    )
  }

  return (
    <section className="mt-12 border-t border-border pt-6">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="inline-flex items-center gap-2 rounded-full border border-border px-3 py-1.5 text-sm text-fg-muted transition-colors hover:border-accent hover:text-accent"
      >
        <CommentIcon />
        <span>评论{total > 0 ? ` ${total}` : ''}</span>
        <span className={`transition-transform ${open ? 'rotate-180' : ''}`}>▾</span>
      </button>

      {open && (
        <div className="mt-5">
          <CommentList slug={slug} />
          <CommentForm slug={slug} />
        </div>
      )}
    </section>
  )
}

/** 历史评论列表 + 滚动分页加载。 */
function CommentList({ slug }: { slug: string }) {
  const {
    data,
    isPending,
    isError,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useComments(slug, true)
  const sentinel = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const el = sentinel.current
    if (!el) return
    const obs = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { rootMargin: '200px' },
    )
    obs.observe(el)
    return () => obs.disconnect()
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  if (isPending) {
    return <div className="h-20 animate-pulse rounded border border-border bg-card" />
  }
  if (isError) {
    return <p className="text-sm text-fg-muted">评论加载失败：{(error as Error).message}</p>
  }

  const comments: Comment[] = data?.pages.flatMap((p) => p.items) ?? []
  const total = data?.pages[0]?.total ?? 0

  if (comments.length === 0) {
    return <p className="mb-6 text-sm text-fg-muted">还没有评论，来说两句吧。</p>
  }

  return (
    <>
      <ul className="mb-8 flex flex-col gap-4">
        {comments.map((c) => (
          <li key={c.public_id} className="rounded-lg border border-border bg-card p-4">
            <div className="mb-1 flex items-center gap-2 text-sm">
              <span className="font-medium">{c.author_name}</span>
              <span className="text-fg-muted">{formatDate(c.created_at)}</span>
            </div>
            <p className="whitespace-pre-wrap text-sm leading-relaxed">{c.content}</p>
          </li>
        ))}
      </ul>

      {/* 触底哨兵：进入视口就加载下一页 */}
      <div ref={sentinel} className="h-8" />
      {isFetchingNextPage && (
        <p className="py-2 text-center text-sm text-fg-muted">加载更多…</p>
      )}
      {!hasNextPage && (
        <p className="py-2 text-center text-sm text-fg-muted">没有更多评论了</p>
      )}
      {hasNextPage && !isFetchingNextPage && comments.length < total && (
        <p className="py-2 text-center text-sm text-fg-muted">向下滚动加载更多</p>
      )}
    </>
  )
}

/** 提交新评论：无审核，提交即通过（蜜罐/限流在服务端拦截）。
 *
 * 已登录用户：昵称固定为 display_name，不可手动修改；匿名用户仍可自定义昵称。
 */
function CommentForm({ slug }: { slug: string }) {
  const qc = useQueryClient()
  const user = useAuth((s) => s.user)
  const isAuthed = useAuth((s) => s.status === 'authed')
  const fixedName = isAuthed && user ? (user.display_name || user.email.split('@')[0]) : ''
  const [name, setName] = useState('')
  const [content, setContent] = useState('')
  const [hp, setHp] = useState('') // 蜜罐：必须留空
  const [error, setError] = useState<string | null>(null)
  const [justPosted, setJustPosted] = useState(false)

  const submit = useMutation({
    mutationFn: () =>
      commentsApi.submit(slug, {
        author_name: isAuthed ? fixedName : (name.trim() || '匿名'),
        content: content.trim(),
        hp,
      }),
    onSuccess: () => {
      setContent('')
      if (!isAuthed) setName('')
      setJustPosted(true)
      setError(null)
      qc.invalidateQueries({ queryKey: ['comments', slug] })
      qc.invalidateQueries({ queryKey: ['comment-count', slug] })
    },
    onError: (err) => {
      setError(err instanceof ApiError ? err.message : '提交失败，请稍后再试')
    },
  })

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        if (!content.trim()) return
        setJustPosted(false)
        submit.mutate()
      }}
      className="flex flex-col gap-3"
    >
      {isAuthed ? (
        <div className="rounded-md border border-border bg-bg-subtle px-3 py-2 text-sm text-fg-muted">
          以 <span className="font-medium text-fg">{fixedName}</span> 的身份发表评论
        </div>
      ) : (
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="昵称（可选）" className="input max-w-xs" />
      )}
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="写下你的评论…"
        rows={3}
        required
        className="input resize-y"
      />
      {/* 蜜罐字段：人类看不到也不会填，机器人可能填 */}
      <input
        value={hp}
        onChange={(e) => setHp(e.target.value)}
        tabIndex={-1}
        autoComplete="off"
        className="hidden"
        aria-hidden
      />
      {error && <p className="text-sm text-red-600 dark:text-red-300">{error}</p>}
      {justPosted && !error && (
        <p className="text-sm text-green-600 dark:text-green-300">评论已发布，感谢参与！</p>
      )}
      <button
        type="submit"
        disabled={submit.isPending}
        className="self-start rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:opacity-90 disabled:opacity-60"
      >
        {submit.isPending ? '提交中…' : '发表评论'}
      </button>
    </form>
  )
}
