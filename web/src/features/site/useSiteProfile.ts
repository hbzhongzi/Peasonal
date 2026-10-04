// 首页身份区的数据源，按「视角」分两套：
//
//   站长视角 useOwnerProfile() —— 关于页等「介绍本站博主」的场景，不随登录态变化，
//       数据来自匿名接口 /site/profile（站长自己在「个人资料」页维护）。
//   当前视角 useViewerProfile() —— 登录后身份区跟随**当前登录用户**（资料直接取自
//       auth store 里的会话数据，零额外请求）；匿名访客回落到站长视角。
//       首页 Hero、侧边栏个人卡片、顶栏品牌、页脚都用它——换个人登录，
//       「你好，我是 XX」「XX的博客」和统计数字就该换成那个人的。
//
// site.ts 常量退化为兜底默认值：后端没配或请求失败时页面不至于开天窗。
import { useQuery } from '@tanstack/react-query'
import { site, type SocialLink } from '@/config/site'
import { authApi } from '@/features/auth/api'
import { useAuth } from '@/features/auth/store'

export interface SiteIdentity {
  /** 昵称 */
  name: string
  /** 一句话简介 */
  tagline: string
  /** 头像地址，空串表示用首字母占位 */
  avatarUrl: string
  /** GitHub 主页，空串表示不展示 */
  github: string
  /** true = 登录者视角（本人会话数据）；false = 站长视角（/site/profile） */
  isMe: boolean
  /** 登录者的对外标识（仅 isMe 时有值）：「我的主页」链接用 */
  publicId?: string
  /** 登录者自己的邮箱（仅 isMe 时有值）：Email 按钮用。展示给本人自己没有隐私问题 */
  email?: string
}

export const siteProfileQuery = {
  queryKey: ['site-profile'] as const,
  queryFn: () => authApi.siteProfile(),
  // 站长资料改动频率极低，缓存 5 分钟足够；失败不重试（匿名接口，别刷屏）
  staleTime: 5 * 60 * 1000,
  retry: false,
}

/** 站长视角（不随登录态变化）：关于页用 */
export function useOwnerProfile(): SiteIdentity {
  const { data } = useQuery(siteProfileQuery)
  return {
    name: data?.display_name?.trim() || site.name,
    tagline: data?.bio?.trim() || site.tagline,
    avatarUrl: data?.avatar_url?.trim() || site.avatar,
    github: data?.github?.trim() || '',
    isMe: false,
  }
}

/** 当前视角：登录 → 本人；匿名 → 站长。首页身份区、顶栏品牌、页脚用 */
export function useViewerProfile(): SiteIdentity {
  const status = useAuth((s) => s.status)
  const user = useAuth((s) => s.user)
  const owner = useOwnerProfile()

  if (status === 'authed' && user) {
    return {
      name: user.display_name?.trim() || '我',
      tagline: user.bio?.trim() || '这个人还没有写简介',
      avatarUrl: user.avatar_url?.trim() || '',
      github: user.github?.trim() || '',
      isMe: true,
      publicId: user.public_id,
      email: user.email,
    }
  }
  return owner
}

/** 我的视角发布统计（登录态首页身份区展示本人的 文章/标签/写作月 数字） */
export const myStatsQuery = {
  queryKey: ['me-stats'] as const,
  queryFn: () => authApi.myStats(),
  staleTime: 60 * 1000,
}

export function useMyStats(enabled: boolean) {
  return useQuery({ ...myStatsQuery, enabled })
}

/** 组社交按钮：GitHub 用当前视角的 github；登录者的 Email 用自己邮箱，匿名回落站点常量 */
export function socialsFor(github: string, email?: string): SocialLink[] {
  if (email) {
    const list: SocialLink[] = []
    if (github) list.push({ label: 'GitHub', href: github, badge: 'GH' })
    list.push({ label: 'Email', href: `mailto:${email}`, badge: '@' })
    return list
  }
  // 站长视角：GitHub 优先用资料里的，其余沿用常量
  if (!github) return site.socials
  return [
    { label: 'GitHub', href: github, badge: 'GH' },
    ...site.socials.filter((s) => s.label !== 'GitHub'),
  ]
}
