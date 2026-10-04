import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { marked } from 'marked'
import { adminPostsApi, type PostWriteInput, type AdminPostDetail } from '../adminApi'
import { mediaApi, type MediaItem } from '@/features/media/api'
import { ApiError } from '@/lib/api'

const EMPTY = {
  title: '',
  slug: '',
  summary: '',
  content_md: '',
  cover_url: '',
  comment_enabled: true,
  tags: [] as string[],
  publish: false,
}

export function PostEditorPage() {
  const { id } = useParams()
  const isEdit = Boolean(id)
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [form, setForm] = useState({ ...EMPTY })
  const [tagsText, setTagsText] = useState('')
  const [showPreview, setShowPreview] = useState(false)
  const [showMedia, setShowMedia] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [saveTime, setSaveTime] = useState<string | null>(null)

  // 编辑态：拉取已有文章
  const existing = useQuery({
    queryKey: ['admin-post', id],
    queryFn: () => adminPostsApi.get(id!),
    enabled: isEdit,
  })

  useEffect(() => {
    if (existing.data) {
      setForm({
        title: existing.data.title,
        slug: existing.data.slug,
        summary: existing.data.summary,
        content_md: existing.data.content_md,
        cover_url: existing.data.cover_url,
        comment_enabled: existing.data.comment_enabled,
        tags: existing.data.tags,
        publish: existing.data.status === 2,
      })
      setTagsText(existing.data.tags.join(', '))
    }
  }, [existing.data])

  function set<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((f) => ({ ...f, [key]: value }))
    setSaved(false)
    setError(null)
  }

  const save = useMutation({
    mutationFn: (publish: boolean) => {
      const body: PostWriteInput = { ...form, tags: parseTags(tagsText), publish }
      if (!body.title.trim()) throw new ApiError('invalid', '请填写标题')
      return isEdit
        ? adminPostsApi.update(id!, body)
        : adminPostsApi.create(body)
    },
    onSuccess: (savedPost: AdminPostDetail) => {
      setSaved(true)
      setSaveTime(new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }))
      setError(null)
      syncForm(savedPost)
      qc.invalidateQueries({ queryKey: ['admin-posts'] })
      if (isEdit) {
        qc.invalidateQueries({ queryKey: ['admin-post', id] })
      } else {
        navigate(`/studio/posts/${savedPost.public_id}`, { replace: true })
      }
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : '保存失败'
      setError(msg)
      // eslint-disable-next-line no-console
      console.error('保存文章失败:', err)
    },
  })

  function syncForm(post: AdminPostDetail) {
    setForm({
      title: post.title,
      slug: post.slug,
      summary: post.summary,
      content_md: post.content_md,
      cover_url: post.cover_url,
      comment_enabled: post.comment_enabled,
      tags: post.tags,
      publish: post.status === 2,
    })
    setTagsText(post.tags.join(', '))
  }

  if (isEdit && existing.isPending) {
    return <div className="h-40 animate-pulse rounded border border-border bg-card" />
  }

  const previewHtml = marked.parse(form.content_md, { async: false }) as string

  return (
    <div>
      <div className="mb-5 flex items-center justify-between gap-3">
        <h1 className="text-xl font-bold">{isEdit ? '编辑文章' : '写新文章'}</h1>
        <div className="flex items-center gap-2">
          {saved && saveTime && (
            <span className="text-xs text-green-600 dark:text-green-300">
              已于 {saveTime} 保存
            </span>
          )}
          <button
            onClick={() => save.mutate(false)}
            disabled={save.isPending}
            className="rounded-md border border-border px-4 py-2 text-sm text-fg-muted hover:border-accent hover:text-accent disabled:opacity-60"
          >
            {save.isPending ? '保存中…' : '保存草稿'}
          </button>
          <button
            onClick={() => save.mutate(true)}
            disabled={save.isPending}
            className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:opacity-90 disabled:opacity-60"
          >
            {save.isPending ? '保存中…' : form.publish ? '更新并发布' : '发布'}
          </button>
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300">
          {error}
        </p>
      )}

      <div className="grid gap-6 lg:grid-cols-[1fr_300px]">
        {/* 左：标题 + 正文 */}
        <div className="flex flex-col gap-4">
          <input
            value={form.title}
            onChange={(e) => set('title', e.target.value)}
            placeholder="标题"
            className="w-full rounded-md border border-border bg-card px-3 py-2 text-lg font-semibold outline-none focus:border-accent"
          />

          <div className="flex items-center gap-2 text-sm">
            <button
              onClick={() => setShowPreview((v) => !v)}
              className={`rounded-md border px-3 py-1.5 ${
                showPreview ? 'border-accent text-accent' : 'border-border text-fg-muted'
              }`}
            >
              {showPreview ? '编辑' : '预览'}
            </button>
            <span className="text-fg-muted">支持 Markdown</span>
          </div>

          {showPreview ? (
            <div
              className="prose-custom min-h-[400px] rounded-md border border-border bg-card p-4"
              dangerouslySetInnerHTML={{ __html: previewHtml }}
            />
          ) : (
            <textarea
              value={form.content_md}
              onChange={(e) => set('content_md', e.target.value)}
              placeholder="在这里用 Markdown 写正文…"
              className="min-h-[420px] w-full resize-y rounded-md border border-border bg-card px-3 py-2 font-mono text-sm leading-relaxed outline-none focus:border-accent"
            />
          )}
        </div>

        {/* 右：元数据 */}
        <aside className="flex flex-col gap-4">
          <Field label="Slug（留空自动生成）">
            <input
              value={form.slug}
              onChange={(e) => set('slug', e.target.value)}
              placeholder="my-post"
              className="input"
            />
          </Field>

          <Field label="摘要">
            <textarea
              value={form.summary}
              onChange={(e) => set('summary', e.target.value)}
              rows={3}
              placeholder="用于列表/分享卡片，留空自动截取正文"
              className="input resize-y"
            />
          </Field>

          <Field label="标签（逗号分隔）">
            <input
              value={tagsText}
              onChange={(e) => {
                setTagsText(e.target.value)
                setSaved(false)
                setError(null)
              }}
              placeholder="前端, Go, 随笔"
              className="input"
            />
          </Field>

          <Field label="封面图">
            <div className="flex items-center gap-2">
              <input
                value={form.cover_url}
                onChange={(e) => set('cover_url', e.target.value)}
                placeholder="https://… 或 /uploads/…"
                className="input flex-1"
              />
              <button
                onClick={() => setShowMedia(true)}
                className="shrink-0 rounded-md border border-border px-3 py-2 text-sm text-fg-muted hover:border-accent hover:text-accent"
              >
                选择
              </button>
            </div>
            {form.cover_url && (
              <img
                src={form.cover_url}
                alt="封面预览"
                className="mt-2 h-28 w-full rounded-md border border-border object-cover"
              />
            )}
          </Field>

          <label className="flex items-center gap-2 text-sm text-fg-muted">
            <input
              type="checkbox"
              checked={form.comment_enabled}
              onChange={(e) => set('comment_enabled', e.target.checked)}
              className="h-4 w-4 accent-[var(--accent)]"
            />
            允许评论
          </label>
        </aside>
      </div>

      {showMedia && (
        <MediaPicker
          onPick={(url) => {
            set('cover_url', url)
            setShowMedia(false)
          }}
          onClose={() => setShowMedia(false)}
        />
      )}
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1.5 text-sm">
      <span className="text-fg-muted">{label}</span>
      {children}
    </label>
  )
}

function parseTags(text: string): string[] {
  return text
    .split(/[,，]/)
    .map((t) => t.trim())
    .filter(Boolean)
}

// 媒体选择弹窗：列出已上传图片，点击即作为封面。
function MediaPicker({
  onPick,
  onClose,
}: {
  onPick: (url: string) => void
  onClose: () => void
}) {
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['media-picker', page],
    queryFn: () => mediaApi.list(page, 18),
  })

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        className="flex max-h-[80vh] w-full max-w-3xl flex-col rounded-xl border border-border bg-card p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="font-semibold">选择封面图</h3>
          <button onClick={onClose} className="text-fg-muted hover:text-fg">
            ✕
          </button>
        </div>

        {query.isPending && (
          <div className="h-40 animate-pulse rounded border border-border" />
        )}
        {query.data && query.data.items.length === 0 && (
          <p className="py-10 text-center text-sm text-fg-muted">
            暂无图片，请先去「媒体库」上传。
          </p>
        )}

        {query.data && query.data.items.length > 0 && (
          <>
            <div className="grid grid-cols-3 gap-3 overflow-y-auto sm:grid-cols-4">
              {query.data.items.map((m: MediaItem) => (
                <button
                  key={m.public_id}
                  onClick={() => onPick(m.url)}
                  className="group overflow-hidden rounded-md border border-border hover:border-accent"
                >
                  <img
                    src={m.url}
                    alt={m.filename}
                    className="h-24 w-full object-cover"
                  />
                </button>
              ))}
            </div>
            {query.data.total > query.data.page_size && (
              <div className="mt-4 flex justify-center gap-2 text-sm">
                <button
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  className="rounded border border-border px-3 py-1 disabled:opacity-40"
                >
                  上一页
                </button>
                <button
                  disabled={page * query.data.page_size >= query.data.total}
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
    </div>
  )
}
