// 站点与博主的唯一配置。
//
// 为什么放前端常量而不是数据库：博主身份信息改动频率以「年」计，
// 为它建表、做后台编辑（M2 的表单要多一页）不值当 —— 等真出现
// "经常改简介" 的需求，再挪进 users 表 / settings 表，改动面只有这一个文件。
//
// Hero（首页顶部）、关于页、侧边栏个人卡片、页脚，全部从这取值。
// 想换名字、换链接，只改这里。

export interface SocialLink {
  label: string
  href: string
  /** 简单的文字缩写，渲染成圆形小徽章；不引入图标库，避免为 3 个链接加几十 KB */
  badge: string
}

export const site = {
  /** 博主显示名。Avatar 组件会用它的第一个字符做默认头像 */
  name: '老麦',
  /** 首页 Hero 与侧边栏的一句话简介 */
  tagline: '后端开发，写 Go，也写点让人看得懂的字。',
  /** 关于页的长介绍，支持换行 */
  bio: [
    '后端开发，日常和 Go、MySQL、消息队列打交道。相信写得清楚比写得炫重要，所以这个博客里的大部分文章，都是在把「我踩过的坑」翻译成「你不用踩」。',
    '下班后喜欢折腾自己的小工具——这个博客本身就是其中一个：Go 渲染 Markdown，React 负责脸面，跑在一台 4 核 4G 的小机器上。',
  ],
  /** 头像图片地址。留空则渲染「首字母 + 渐变」的占位头像 */
  avatar: '',
  /** 头像图片地址为空时，渐变占位头像的两个颜色（任意 CSS 颜色） */
  avatarFrom: 'oklch(72% 0.15 260)',
  avatarTo: 'oklch(65% 0.17 200)',
  socials: [
    { label: 'GitHub', href: 'https://github.com', badge: 'GH' },
    { label: 'Email', href: 'mailto:hi@example.com', badge: '@' },
  ] satisfies SocialLink[],
  /** 页脚版权区间从哪一年开始 */
  since: 2026,
  /** 页脚 ICP 备案号。没有就留空字符串，页脚会自动隐藏这一段 */
  icp: '',
} as const
