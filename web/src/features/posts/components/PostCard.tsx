import type { PostListItem } from '../api'
import { formatDate } from '@/lib/format'
import { Link } from 'react-router-dom'

export function PostCard({ post }: { post: PostListItem }) {
  return (
    <article className="group rounded-lg border border-transparent p-4 transition-colors hover:border-border hover:bg-card sm:-mx-4">
      <h2 className="mb-2 text-xl font-semibold">
        <Link
          to={`/posts/${encodeURIComponent(post.slug)}`}
          className="transition-colors group-hover:text-accent"
        >
          {post.title}
        </Link>
      </h2>

      <div className="mb-2 flex flex-wrap items-center text-sm text-fg-muted">
        <time dateTime={post.published_at ?? undefined}>{formatDate(post.published_at)}</time>
        <span className="mx-2">·</span>
        <span>约 {post.reading_time_min} 分钟</span>
        {/* 站上不止一个作者后，文章卡片必须能看出是谁写的；点名字去作者的博客主页 */}
        {post.author?.public_id && (
          <>
            <span className="mx-2">·</span>
            <Link
              to={`/blog/${post.author.public_id}`}
              className="transition-colors hover:text-accent"
            >
              {post.author.display_name || '佚名'}
            </Link>
          </>
        )}
      </div>

      {post.summary && (
        <p className="mb-3 text-sm leading-relaxed text-fg-muted">{post.summary}</p>
      )}

      {post.tags.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {post.tags.map((t) => (
            <Link
              key={t.slug}
              to={`/tags/${encodeURIComponent(t.slug)}`}
              className="rounded-full border border-border bg-bg-subtle px-2.5 py-0.5 text-xs text-fg-muted transition-colors hover:border-accent hover:text-accent"
            >
              {t.name}
            </Link>
          ))}
        </div>
      )}
    </article>
  )
}
