import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { mediaApi, type MediaItem } from '../api'
import { ApiError } from '@/lib/api'
import { formatBytes } from '@/lib/format'
import { useAuth } from '@/features/auth/store'

export function MediaLibraryPage() {
  const [page, setPage] = useState(1)
  const [msg, setMsg] = useState<string | null>(null)
  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)
  // 上传是管理员授予的特权：没有该权限时，媒体库只可浏览、不可上传
  const canUpload = useAuth((s) => s.canUpload())

  const query = useQuery({
    queryKey: ['media', page],
    queryFn: () => mediaApi.list(page, 24),
  })

  const uploadMut = useMutation({
    mutationFn: (file: File) => mediaApi.upload(file),
    onSuccess: () => {
      setMsg('上传成功')
      qc.invalidateQueries({ queryKey: ['media'] })
    },
    onError: (err) => {
      const m = err instanceof ApiError ? err.message : '上传失败'
      setMsg(m)
    },
  })

  const removeMut = useMutation({
    mutationFn: (id: string) => mediaApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['media'] }),
  })

  function onPick(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) {
      setMsg(null)
      uploadMut.mutate(file)
    }
    e.target.value = ''
  }

  const data = query.data

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-xl font-bold">媒体库</h1>
        <div>
          <input
            ref={fileRef}
            type="file"
            accept="image/png,image/jpeg,image/gif,image/webp"
            className="hidden"
            onChange={onPick}
          />
          {canUpload ? (
            <button
              onClick={() => fileRef.current?.click()}
              disabled={uploadMut.isPending}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:opacity-90 disabled:opacity-60"
            >
              {uploadMut.isPending ? '上传中…' : '上传图片'}
            </button>
          ) : (
            <span
              title="上传图片为管理员授予的特权，请联系管理员开启"
              className="rounded-md border border-border px-4 py-2 text-sm text-fg-muted"
            >
              无上传权限
            </span>
          )}
        </div>
      </div>

      {!canUpload && (
        <p className="mb-4 rounded-md border border-border bg-card px-3 py-2 text-sm text-fg-muted">
          你的账号暂无图片上传权限。可在文章正文中使用外链图片，或联系管理员为你开启上传权限。
        </p>
      )}

      {msg && (
        <p className="mb-4 rounded-md border border-border bg-card px-3 py-2 text-sm text-fg-muted">
          {msg}
        </p>
      )}

      {query.isPending && (
        <div className="h-40 animate-pulse rounded border border-border bg-card" />
      )}
      {query.isError && (
        <p className="rounded border border-border bg-card p-4 text-sm text-red-600 dark:text-red-300">
          {query.error instanceof ApiError ? query.error.message : '加载失败'}
        </p>
      )}

      {data && data.items.length === 0 && (
        <p className="rounded border border-border bg-card p-6 text-center text-sm text-fg-muted">
          还没有图片，点击右上角「上传图片」开始。（支持 PNG/JPEG/GIF/WebP，超过 1600px 宽会自动缩放）
        </p>
      )}

      {data && data.items.length > 0 && (
        <>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
            {data.items.map((m: MediaItem) => (
              <div
                key={m.public_id}
                className="group overflow-hidden rounded-lg border border-border bg-card"
              >
                <div className="relative">
                  <img
                    src={m.url}
                    alt={m.filename}
                    className="h-32 w-full object-cover"
                  />
                  <button
                    onClick={() => {
                      if (confirm(`删除这张图片？`)) removeMut.mutate(m.public_id)
                    }}
                    className="absolute right-2 top-2 hidden rounded bg-black/60 px-2 py-1 text-xs text-white group-hover:block hover:bg-black/80"
                  >
                    删除
                  </button>
                </div>
                <div className="px-3 py-2 text-xs">
                  <p className="truncate text-fg" title={m.filename}>
                    {m.filename}
                  </p>
                  <p className="mt-0.5 text-fg-muted">
                    {formatBytes(m.size_bytes)}
                    {m.width > 0 && ` · ${m.width}×${m.height}`}
                  </p>
                  <button
                    onClick={() => navigator.clipboard?.writeText(m.url)}
                    className="mt-1 text-accent hover:underline"
                  >
                    复制链接
                  </button>
                </div>
              </div>
            ))}
          </div>

          {data.total > data.page_size && (
            <div className="mt-6 flex justify-center gap-2 text-sm">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="rounded border border-border px-3 py-1 disabled:opacity-40"
              >
                上一页
              </button>
              <span className="px-2 py-1 text-fg-muted">
                {page} / {Math.ceil(data.total / data.page_size)}
              </span>
              <button
                disabled={page * data.page_size >= data.total}
                onClick={() => setPage((p) => p + 1)}
                className="rounded border border-border px-3 py-1 disabled:opacity-40"
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
