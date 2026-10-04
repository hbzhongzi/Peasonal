// 媒体库（本地磁盘图片）接口。
import { request, upload } from '@/lib/api'
import type { Page } from '@/features/posts/api'

export interface MediaItem {
  public_id: string
  url: string
  filename: string
  size_bytes: number
  mime_type: string
  width: number
  height: number
  created_at: string
}

export const mediaApi = {
  list: (page = 1, pageSize = 24) =>
    request<Page<MediaItem>>('GET', '/admin/media', undefined, { page, page_size: pageSize }),

  remove: (id: string) => request<void>('DELETE', `/admin/media/${id}`),

  // 上传：multipart，字段名 file
  upload: (file: File) => upload<MediaItem>('/admin/media', file, 'file'),
}
