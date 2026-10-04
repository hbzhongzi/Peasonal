import { useEffect } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/queryClient'
import { ThemeProvider } from './theme'
import { useAuth } from '@/features/auth/store'
import type { ReactNode } from 'react'

export function Providers({ children }: { children: ReactNode }) {
  const bootstrap = useAuth((s) => s.bootstrap)

  // 应用启动：尝试用 refresh cookie 续期，决定初始是 authed 还是 anon。
  useEffect(() => {
    void bootstrap()
  }, [bootstrap])

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>{children}</ThemeProvider>
    </QueryClientProvider>
  )
}
