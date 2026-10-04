import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { usersApi } from '../api'
import { PostList } from '@/features/posts/components/PostList'
import { Pagination } from '@/features/posts/components/Pagination'
import { Avatar } from '@/components/Avatar'
import { ApiError } from '@/lib/api'
import { useAuth } from '@/features/auth/store'

// 用户博客主页（/blog/:publicID）：
// 「不同用户的相关界面」的公开侧 —— 每个注册用户都有自己的一个页面，
// 展示他的公开资料（昵称/简介/头像/GitHub）与他发布的文章。
// 作者的草稿不出现在这里：公开接口只列 status=2。
//
// 本人查看自己的主页时，资料头下方出现「写新文章 / 管理我的博客」——
// 这就是「普通用户的个人主页也能编辑或发布文章」的落点。它只对自己可见，
// 属于对自己内容的操作，不是把后台入口暴露在公开导航里。
export function UserBlogPage() {
  const { publicID = '' } = useParams()
  // 分页走 URL（项目约定）：可分享、可后退
  const [searchParams] = useSearchParams()
  const page = Number(searchParams.get('page')) || 1

  // 本人查看自己的主页？这两条来自登录态，决定要不要显示编辑/发布入口
  const status = useAuth((s) => s.status)
  const me = useAuth((s) => s.user)
  const canPublish = useAuth((s) => s.canPublish())
  const isOwner = status === 'authed' && !!me && me.public_id === publicID

  const profile = useQuery({
    queryKey: ['user-profile', publicID],
    queryFn: () => usersApi.profile(publicID),
    enabled: publicID.length > 0,
    retry: false,
  })
  const posts = useQuery({
    queryKey: ['user-posts', publicID, page],
    queryFn: () => usersApi.posts(publicID, page, 10),
    enabled: publicID.length > 0,
  })

  if (profile.isError) {
    const notFound = profile.error instanceof ApiError && profile.error.message.includes('不存在')
    return (
      <div className="mx-auto max-w-2xl rounded border border-border bg-card p-8 text-center">
        <p className="font-semibold">{notFound ? '用户不存在' : '加载失败'}</p>
        <Link to="/" className="mt-4 inline-block text-accent">
          返回首页
        </Link>
      </div>
    )
  }

  if (profile.isPending) {
    return <div className="h-48 animate-pulse rounded-xl border border-border bg-card" />
  }

  const p = profile.data

  return (
    <div className="mx-auto max-w-3xl">
      {/* 用户资料头：与首页 Hero 同一种语言，但数据来自这个用户本人 */}
      <section className="mb-8 rounded-xl border border-border bg-card p-6 sm:p-8">
        <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
          <Avatar size="lg" name={p.display_name} avatarUrl={p.avatar_url} />
          <div className="min-w-0">
            <h1 className="text-2xl font-bold sm:text-3xl">{p.display_name}的博客</h1>
            {p.bio && <p className="mt-1.5 text-fg-muted">{p.bio}</p>}
          </div>
        </div>
        {p.github && (
          <div className="mt-5 flex flex-wrap gap-2">
            <a
              href={p.github}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-bg-subtle px-3.5 py-1 text-sm text-fg-muted transition-colors hover:border-accent hover:text-accent"
            >
              <span className="text-xs font-semibold">GH</span>
              GitHub
            </a>
          </div>
        )}

        {/* 本人查看自己的主页：给出写/管文章的入口（仅对本人、且仅当拥有发帖权限时显示） */}
        {isOwner && canPublish && (
          <div className="mt-5 flex flex-wrap gap-3">
            <Link
              to="/studio/posts/new"
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:opacity-90"
            >
              写新文章
            </Link>
            <Link
              to="/studio"
              className="rounded-md border border-border px-4 py-2 text-sm text-fg-muted transition-colors hover:border-accent hover:text-accent"
            >
              管理我的博客
            </Link>
          </div>
        )}
      </section>

      {posts.isPending ? (
        <div className="h-40 animate-pulse rounded border border-border bg-card" />
      ) : posts.isError ? (
        <p className="rounded border border-border bg-card p-4 text-sm text-red-600 dark:text-red-300">
          文章加载失败
        </p>
      ) : (
        <>
          <PostList items={posts.data.items} />
          <Pagination
            page={page}
            total={posts.data.total}
            pageSize={posts.data.page_size}
            to={(next) => `/blog/${publicID}?page=${next}`}
          />
        </>
      )}
    </div>
  )
}
