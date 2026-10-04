import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

export type ThemeMode = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'blog-theme'
const DARK_QUERY = '(prefers-color-scheme: dark)'

type ThemeContextValue = {
  mode: ThemeMode
  resolved: ResolvedTheme
  setMode: (mode: ThemeMode) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function readStoredMode(): ThemeMode {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    /* localStorage 不可用（隐私模式）时忽略 */
  }
  return 'system'
}

function systemPrefersDark(): boolean {
  return window.matchMedia(DARK_QUERY).matches
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  // 初值直接同步读取，避免首帧与 index.html 里那段脚本的结果不一致
  const [mode, setModeState] = useState<ThemeMode>(readStoredMode)
  const [systemDark, setSystemDark] = useState<boolean>(systemPrefersDark)

  // 跟随系统：监听系统主题变化
  useEffect(() => {
    const mq = window.matchMedia(DARK_QUERY)
    const onChange = (e: MediaQueryListEvent) => setSystemDark(e.matches)
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])

  const resolved: ResolvedTheme =
    mode === 'system' ? (systemDark ? 'dark' : 'light') : mode

  // 把结果落到 <html> 上。index.html 里的内联脚本负责首屏，这里负责后续切换。
  useEffect(() => {
    const root = document.documentElement
    root.classList.toggle('dark', resolved === 'dark')
    root.style.colorScheme = resolved
  }, [resolved])

  const setMode = useCallback((next: ThemeMode) => {
    setModeState(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      /* 忽略写入失败 */
    }
  }, [])

  const value = useMemo(
    () => ({ mode, resolved, setMode }),
    [mode, resolved, setMode],
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme 必须在 ThemeProvider 内部使用')
  return ctx
}
