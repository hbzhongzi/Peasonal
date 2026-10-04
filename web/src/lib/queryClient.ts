import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 内容型数据 5 分钟内视为新鲜，避免每次路由切换都打接口
      staleTime: 5 * 60 * 1000,
      // 失败重试 1 次即可，内容站不需要激进重试
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})
