import { useTheme, type ThemeMode } from '@/app/theme'

const OPTIONS: { value: ThemeMode; label: string }[] = [
  { value: 'light', label: '亮色' },
  { value: 'dark', label: '暗色' },
  { value: 'system', label: '跟随系统' },
]

export function ThemeToggle() {
  const { mode, resolved, setMode } = useTheme()

  return (
    <div
      style={{
        display: 'inline-flex',
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
      }}
    >
      {OPTIONS.map((opt) => {
        const active = mode === opt.value
        return (
          <button
            key={opt.value}
            type="button"
            onClick={() => setMode(opt.value)}
            aria-pressed={active}
            style={{
              padding: '4px 10px',
              fontSize: 13,
              border: 0,
              cursor: 'pointer',
              background: active ? 'var(--accent)' : 'transparent',
              color: active ? 'var(--accent-fg)' : 'var(--fg-muted)',
            }}
          >
            {opt.label}
          </button>
        )
      })}
      <span
        style={{
          padding: '4px 10px',
          fontSize: 12,
          color: 'var(--fg-muted)',
          alignSelf: 'center',
        }}
      >
        当前：{resolved === 'dark' ? '暗' : '亮'}
      </span>
    </div>
  )
}
