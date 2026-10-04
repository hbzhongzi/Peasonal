import { useRef } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Toc } from '../components/Toc'
import { useCodeCopy } from '../hooks/useCodeCopy'
import { usePostDetail } from '../hooks'
import { formatDate } from '@/lib/format'
import { Avatar } from '@/components/Avatar'
import { CommentSection } from '@/features/comments/components/CommentSection'

export function PostDetailPage() {
  const { slug = '' } = useParams()
  const { data, isPending, isError, error } = usePostDetail(slug)

  const bodyRef = useRef<HTMLDivElement>(null)
  useCodeCopy(bodyRef, [data?.content_html])

  if (isError) {
    const notFound = error instanceof Error && error.message.includes('不存在')
    return (
      <div className="rounded border border-border bg-card p-6">
        <p className="font-semibold">{notFound ? '文章不存在' : '加载失败'}</p>
        <p className="mt-1 text-sm text-fg-muted">{error.message}</p>
        <Link to="/" className="mt-4 inline-block text-accent">
          返回首页
        </Link>
      </div>
    )
  }

  if (isPending) {
    return <div className="h-64 animate-pulse rounded border border-border bg-card" />
  }

  return (
    <article className="mx-auto max-w-3xl">
      <header className="mb-6 border-b border-border pb-4">
        <div className="flex items-start justify-between gap-4">
          <h1 className="mb-2 text-3xl font-bold">{data.title}</h1>
        </div>
        <div className="flex flex-wrap items-center text-sm text-fg-muted">
          <time dateTime={data.published_at ?? undefined}>{formatDate(data.published_at)}</time>
          <span className="mx-2">·</span>
          <span>约 {data.reading_time_min} 分钟</span>
          {/* 作者署名：点名可跳到作者的博客主页 */}
          {data.author?.public_id && (
            <>
              <span className="mx-2">·</span>
              <span className="inline-flex items-center gap-1.5">
                <Avatar size="sm" name={data.author.display_name} avatarUrl={data.author.avatar_url} />
                <Link
                  to={`/blog/${data.author.public_id}`}
                  className="transition-colors hover:text-accent"
                >
                  {data.author.display_name || '佚名'}
                </Link>
              </span>
            </>
          )}
        </div>
        {data.tags.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {data.tags.map((t) => (
              <Link
                key={t.slug}
                to={`/tags/${encodeURIComponent(t.slug)}`}
                className="rounded-full border border-border px-2.5 py-0.5 text-xs text-fg-muted hover:border-accent hover:text-accent"
              >
                {t.name}
              </Link>
            ))}
          </div>
        )}
      </header>

      <div className="flex flex-col gap-8 lg:flex-row-reverse lg:items-start">
        {data.toc.length > 1 && (
          <aside className="lg:sticky lg:top-6 lg:w-56 lg:shrink-0">
            <Toc items={data.toc} />
          </aside>
        )}

        {/*
          正文是后端渲染并经过 bluemonday 消毒的 HTML（见技术方案 §7.1）。
          这里不做任何二次拼装 —— 任何字符串拼接都可能重新引入 XSS。
        */}
        <div
          ref={bodyRef}
          className="prose-custom min-w-0 flex-1"
          dangerouslySetInnerHTML={{ __html: data.content_html }}
        />
      </div>

      <CommentSection slug={data.slug} enabled={data.comment_enabled} />
    </article>
  )
}
