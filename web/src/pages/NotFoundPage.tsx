import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div>
      <h1 style={{ fontSize: 24, marginBottom: 8 }}>404</h1>
      <p style={{ color: 'var(--fg-muted)' }}>
        页面不存在。<Link to="/">返回首页</Link>
      </p>
    </div>
  )
}
