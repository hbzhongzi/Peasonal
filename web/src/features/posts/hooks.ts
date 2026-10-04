import { useQuery } from '@tanstack/react-query'
import { postsApi } from './api'

// queryKey 统一在这里定义：散落在组件里容易出现「改了参数但缓存没失效」
export const postKeys = {
  list: (page: number, pageSize: number) => ['posts', 'list', page, pageSize] as const,
  detail: (slug: string) => ['posts', 'detail', slug] as const,
  tags: () => ['tags'] as const,
  byTag: (slug: string, page: number, pageSize: number) =>
    ['tags', slug, 'posts', page, pageSize] as const,
  archive: () => ['archive'] as const,
  byMonth: (year: number, month: number, page: number, pageSize: number) =>
    ['archive', year, month, page, pageSize] as const,
}

export function usePostList(page: number, pageSize = 10) {
  return useQuery({
    queryKey: postKeys.list(page, pageSize),
    queryFn: () => postsApi.list(page, pageSize),
    // 翻页时保留上一页数据，避免列表整块闪成骨架屏
    placeholderData: (prev) => prev,
  })
}

export function usePostDetail(slug: string) {
  return useQuery({
    queryKey: postKeys.detail(slug),
    queryFn: () => postsApi.detail(slug),
    enabled: !!slug,
  })
}

export function useTags() {
  return useQuery({ queryKey: postKeys.tags(), queryFn: () => postsApi.tags() })
}

export function usePostsByTag(slug: string, page: number, pageSize = 10) {
  return useQuery({
    queryKey: postKeys.byTag(slug, page, pageSize),
    queryFn: () => postsApi.byTag(slug, page, pageSize),
    enabled: !!slug,
    placeholderData: (prev) => prev,
  })
}

export function useArchive() {
  return useQuery({ queryKey: postKeys.archive(), queryFn: () => postsApi.archive() })
}

export function usePostsByMonth(year: number, month: number, page: number, pageSize = 10) {
  return useQuery({
    queryKey: postKeys.byMonth(year, month, page, pageSize),
    queryFn: () => postsApi.byMonth(year, month, page, pageSize),
    placeholderData: (prev) => prev,
  })
}
