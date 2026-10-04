import { site } from '@/config/site'

/**
 * 头像组件。
 *
 * 有图片地址就用图片；没有就用「首字母 + 渐变圆」占位——
 * 这比放一张 broken image 或引入第三方头像服务都更省心。
 *
 * name / avatarUrl 可从外部传入（前台传站长资料、工作台传当前登录者），
 * 不传则用 config/site.ts 的兜底值。
 */
export function Avatar({
  size = 'md',
  name,
  avatarUrl,
}: {
  size?: 'sm' | 'md' | 'lg'
  name?: string
  avatarUrl?: string
}) {
  const cls =
    size === 'lg'
      ? 'h-20 w-20 text-3xl'
      : size === 'sm'
        ? 'h-8 w-8 text-sm'
        : 'h-14 w-14 text-xl'

  const displayName = name?.trim() || site.name
  const url = avatarUrl?.trim() || site.avatar

  if (url) {
    return (
      <img
        src={url}
        alt={`${displayName}的头像`}
        className={`${cls} shrink-0 rounded-full border border-border object-cover`}
      />
    )
  }

  return (
    <div
      aria-hidden
      className={`${cls} flex shrink-0 select-none items-center justify-center rounded-full font-bold text-white`}
      style={{
        background: `linear-gradient(135deg, ${site.avatarFrom}, ${site.avatarTo})`,
      }}
    >
      {displayName.slice(0, 1)}
    </div>
  )
}
