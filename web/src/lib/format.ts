// 展示层用到的纯函数。不放业务逻辑，只做格式转换。

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '未发布'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '未发布'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return `${formatDate(iso)} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}
