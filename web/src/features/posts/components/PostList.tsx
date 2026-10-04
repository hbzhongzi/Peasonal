import type { PostListItem } from '../api'
import { PostCard } from './PostCard'

export function PostList({ items }: { items: PostListItem[] }) {
  if (items.length === 0) {
    return (
      <p className="py-12 text-center text-fg-muted">
        还没有已发布的文章。
      </p>
    )
  }
  return (
    <div className="space-y-3">
      {items.map((p) => (
        <PostCard key={p.public_id} post={p} />
      ))}
    </div>
  )
}
