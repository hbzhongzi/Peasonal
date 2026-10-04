import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authApi } from '../api'
import { ApiError } from '@/lib/api'

// 用户管理：列出全部用户，管理员可开启/关闭其发帖与上传权限。
export function UsersAdminPage() {
  const qc = useQueryClient()
  const query = useQuery({
    queryKey: ['users'],
    queryFn: () => authApi.listUsers(),
  })

  const update = useMutation({
    mutationFn: ({ id, canUpload, canPublish, canAdmin }: { id: string; canUpload: boolean; canPublish: boolean; canAdmin: boolean }) =>
      authApi.updatePerms(id, { can_upload: canUpload, can_publish: canPublish, can_admin: canAdmin }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })

  return (
    <div>
      <h1 className="mb-1 text-xl font-bold">用户管理</h1>
      <p className="mb-6 text-sm text-fg-muted">
        注册用户默认即为作者，可在「我的博客」写/发文章；
        <strong className="text-fg">上传图片为管理员授予的特权，默认关闭</strong>，在此按需为信任的用户开启。
        仅站点管理员可进入本页，可在此把某人提升为管理员（拥有后台管理权限）。
      </p>

      {query.isPending && (
        <div className="h-32 animate-pulse rounded border border-border bg-card" />
      )}
      {query.isError && (
        <p className="rounded border border-border bg-card p-4 text-sm text-red-600 dark:text-red-300">
          {query.error instanceof ApiError ? query.error.message : '加载失败'}
        </p>
      )}

      {query.data && query.data.length === 0 && (
        <p className="rounded border border-border bg-card p-6 text-center text-sm text-fg-muted">
          还没有用户。
        </p>
      )}

      {query.data && query.data.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-lg border border-border">
          {query.data.map((u) => (
            <li key={u.id} className="flex items-center gap-4 bg-card px-4 py-3">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{u.display_name || '(无昵称)'}</p>
                <p className="truncate text-xs text-fg-muted">{u.email}</p>
              </div>

              <PermToggle
                label="发帖"
                checked={u.can_publish}
                disabled={update.isPending}
                onChange={(next) =>
                  update.mutate({ id: u.public_id, canUpload: u.can_upload, canPublish: next, canAdmin: u.is_admin })
                }
              />
              <PermToggle
                label="上传"
                checked={u.can_upload}
                disabled={update.isPending}
                onChange={(next) =>
                  update.mutate({ id: u.public_id, canUpload: next, canPublish: u.can_publish, canAdmin: u.is_admin })
                }
              />
              <PermToggle
                label="管理员"
                checked={u.is_admin}
                disabled={update.isPending}
                onChange={(next) =>
                  update.mutate({ id: u.public_id, canUpload: u.can_upload, canPublish: u.can_publish, canAdmin: next })
                }
              />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function PermToggle({
  label,
  checked,
  disabled,
  onChange,
}: {
  label: string
  checked: boolean
  disabled: boolean
  onChange: (next: boolean) => void
}) {
  return (
    <label className="flex shrink-0 items-center gap-1.5 text-sm text-fg-muted">
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 accent-[var(--accent)]"
      />
      {label}
    </label>
  )
}
