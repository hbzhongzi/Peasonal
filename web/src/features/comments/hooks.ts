import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { commentsApi } from './api'

const PAGE_SIZE = 20

/**
 * 文章公开评论的滚动分页：按时间正序，旧评在前、新评在后；
 * 触底时 react-query 自动拉下一页（见 CommentSection 里的 IntersectionObserver）。
 */
export function useComments(slug: string, enabled: boolean) {
  return useInfiniteQuery({
    queryKey: ['comments', slug],
    enabled,
    initialPageParam: 1,
    queryFn: ({ pageParam }) => commentsApi.listPublic(slug, pageParam, PAGE_SIZE),
    getNextPageParam: (lastPage) => {
      const loaded = lastPage.page * lastPage.page_size
      return loaded < lastPage.total ? lastPage.page + 1 : undefined
    },
  })
}

/** 评论图标角标用的总数：不拉内容，轻量查询。 */
export function useCommentCount(slug: string, enabled: boolean) {
  return useQuery({
    queryKey: ['comment-count', slug],
    enabled,
    queryFn: () => commentsApi.count(slug),
    staleTime: 30 * 1000,
  })
}
