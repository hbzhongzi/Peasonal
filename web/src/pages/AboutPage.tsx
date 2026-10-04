import { site } from '@/config/site'
import { Avatar } from '@/components/Avatar'
import { useAuth } from '@/features/auth/store'
import { socialsFor, useOwnerProfile } from '@/features/site/useSiteProfile'

/**
 * 关于页。
 *
 * 用**站长视角**（useOwnerProfile）展示本站博主信息：内容属于站点，不随登录者变化。
 * 只有站长本人看时标题是「关于我」；其他用户（包括已登录的普通作者）看都是「关于作者」。
 * 首页侧边栏与身份区才跟随当前登录用户（见 PostListPage/ProfileCard）。
 * 长篇自我介绍仍是 site.bio 常量（它属于「内容」而非「账号资料」，换人不换文）。
 * 「本站」一节列的是真实技术栈与运行环境（与 docs/implementation-status.md 对应），
 * 改技术栈时记得同步这里 —— 它是给读者看的门面，不是开发文档。
 */
export function AboutPage() {
  const { name, tagline, avatarUrl, github } = useOwnerProfile()
  const socials = socialsFor(github)
  const isOwner = useAuth((s) => s.status === 'authed' && s.user?.is_admin === true)

  return (
    <article className="mx-auto max-w-2xl">
      <header className="flex items-center gap-5">
        <Avatar size="lg" name={name} avatarUrl={avatarUrl} />
        <div>
          <h1 className="text-2xl font-bold">{isOwner ? '关于我' : '关于作者'}</h1>
          <p className="mt-1 text-fg-muted">{tagline}</p>
        </div>
      </header>

      <section className="mt-8 space-y-4 leading-relaxed">
        {site.bio.map((para, i) => (
          <p key={i}>{para}</p>
        ))}
      </section>

      <section className="mt-10">
        <h2 className="text-lg font-semibold">这个站</h2>
        <div className="mt-3 overflow-hidden rounded-lg border border-border">
          <dl className="divide-y divide-border text-sm">
            {[
              ['后端', 'Go · chi · MySQL 8.0 · sqlc + goose'],
              ['前端', 'React 19 · TypeScript · Tailwind CSS v4 · TanStack Query'],
              ['渲染', 'Markdown 在后端渲染成 HTML 落库，代码高亮由 chroma 输出双主题'],
              ['托管', '跑在一台 4 核 4G 的小机器上'],
            ].map(([k, v]) => (
              <div key={k} className="flex gap-4 bg-card px-4 py-2.5">
                <dt className="w-16 shrink-0 text-fg-muted">{k}</dt>
                <dd>{v}</dd>
              </div>
            ))}
          </dl>
        </div>
      </section>

      <section className="mt-10">
        <h2 className="text-lg font-semibold">找到我</h2>
        <ul className="mt-3 space-y-2">
          {socials.map((s) => (
            <li key={s.label}>
              <a
                href={s.href}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 text-accent hover:underline"
              >
                <span className="flex h-6 w-6 items-center justify-center rounded-full border border-border text-[10px] font-semibold">
                  {s.badge}
                </span>
                {s.label}
              </a>
            </li>
          ))}
        </ul>
      </section>
    </article>
  )
}
