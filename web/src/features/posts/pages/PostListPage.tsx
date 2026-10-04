import { useSearchParams } from 'react-router-dom'
import { PostList } from '../components/PostList'
import { Pagination } from '../components/Pagination'
import { useArchive, usePostList, useTags } from '../hooks'
import { ProfileCard } from '@/components/ProfileCard'
import { TagCloud } from '@/components/TagCloud'
import { ArchiveMini } from '@/components/ArchiveMini'

/** 大屏右侧栏：个人卡片 + 标签云 + 最近写作 */
function SideColumn() {
  const { data: tagList } = useTags()
  const { data: archive } = useArchive()
  const { data: one } = usePostList(1, 1)

  const monthTotal = archive?.length
  const tagTotal = tagList?.length

  return (
    <aside className="space-y-5 lg:sticky lg:top-6 lg:self-start">
      <ProfileCard postTotal={one?.total} tagTotal={tagTotal} monthTotal={monthTotal} />

      <section className="rounded-lg border border-border bg-card p-5">
        <h2 className="mb-3 text-sm font-semibold text-fg-muted">标签</h2>
        <TagCloud tags={tagList} max={10} />
      </section>

      <section className="rounded-lg border border-border bg-card p-5">
        <h2 className="mb-3 text-sm font-semibold text-fg-muted">最近写作</h2>
        <ArchiveMini months={archive} />
      </section>
    </aside>
  )
}

export function PostListPage() {
  const [params] = useSearchParams()
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1)

  const { data, isPending, isError, error } = usePostList(page)

  return (
    <div className="grid gap-8 lg:grid-cols-[1fr_280px]">
      <div className="min-w-0">
        {isError ? (
          <div className="rounded border border-border bg-card p-6">
            <p className="font-semibold">文章加载失败</p>
            <p className="mt-1 text-sm text-fg-muted">{error.message}</p>
          </div>
        ) : isPending ? (
          <div className="space-y-4">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-28 animate-pulse rounded-lg border border-border bg-card" />
            ))}
          </div>
        ) : (
          <>
            <PostList items={data.items} />
            <Pagination
              page={data.page}
              pageSize={data.page_size}
              total={data.total}
              to={(next) => `/?page=${next}`}
            />
          </>
        )}
      </div>

      <SideColumn />
    </div>
  )
}
