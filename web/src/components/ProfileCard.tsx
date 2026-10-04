import { Link } from 'react-router-dom'
import { Avatar } from '@/components/Avatar'
import { socialsFor, useMyStats, useViewerProfile } from '@/features/site/useSiteProfile'

/**
 * 侧边栏的个人卡片 —— **视角感知**：
 *
 *   登录时：显示当前用户自己的 资料 + 发布统计（GET /admin/me/stats），
 *           底部链接是「我的主页 →」（/blog/{public_id}）；
 *   匿名时：显示站长资料 + 全站统计（props 传入），链接是「关于我 →」。
 *
 * 换个人登录，这张卡片就该换成那个人的——不再永远显示管理员本人。
 * 统计数字（文章数 / 标签数 / 迄今写作月数）的口径两边一致：只算已发布。
 */
export function ProfileCard({
  postTotal,
  tagTotal,
  monthTotal,
}: {
  postTotal?: number
  tagTotal?: number
  monthTotal?: number
}) {
  const { name, tagline, avatarUrl, github, email, isMe, publicId } = useViewerProfile()
  const { data: my } = useMyStats(isMe)
  const socials = socialsFor(github, email)

  const stats = isMe
    ? [
        { label: '文章', value: my?.post_total },
        { label: '标签', value: my?.tag_total },
        { label: '写作月', value: my?.month_total },
      ]
    : [
        { label: '文章', value: postTotal },
        { label: '标签', value: tagTotal },
        { label: '写作月', value: monthTotal },
      ]

  return (
    <section className="rounded-lg border border-border bg-card p-5">
      <div className="flex items-center gap-4">
        <Avatar name={name} avatarUrl={avatarUrl} />
        <div className="min-w-0">
          <p className="truncate font-semibold">{name}</p>
          <p className="mt-0.5 text-sm leading-snug text-fg-muted">{tagline}</p>
        </div>
      </div>

      <dl className="mt-4 grid grid-cols-3 divide-x divide-border rounded border border-border bg-bg-subtle py-2 text-center">
        {stats.map((s) => (
          <div key={s.label}>
            <dd className="text-base font-semibold tabular-nums">{s.value ?? '–'}</dd>
            <dt className="text-xs text-fg-muted">{s.label}</dt>
          </div>
        ))}
      </dl>

      <div className="mt-4 flex items-center gap-2">
        {socials.map((s) => (
          <a
            key={s.label}
            href={s.href}
            target="_blank"
            rel="noopener noreferrer"
            title={s.label}
            className="flex h-8 w-8 items-center justify-center rounded-full border border-border text-xs font-semibold text-fg-muted transition-colors hover:border-accent hover:text-accent"
          >
            {s.badge}
          </a>
        ))}
        {isMe && publicId ? (
          <Link to={`/blog/${publicId}`} className="ml-auto text-sm text-accent hover:underline">
            我的主页 →
          </Link>
        ) : (
          <Link to="/about" className="ml-auto text-sm text-accent hover:underline">
            关于我 →
          </Link>
        )}
      </div>
    </section>
  )
}
