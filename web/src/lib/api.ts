// 前端对后端的统一入口（公共读 + 带鉴权的写）。
//
// 约定：
//   - 所有请求经 get()/request()/upload()，统一处理超时、HTTP 状态码与 {data,error} 解包。
//   - access token 只存在内存（不在 localStorage，防 XSS 窃取），由 setAuthToken 注入。
//   - refresh token 走 HttpOnly cookie，fetch 带 credentials: 'include' 自动附带。
//   - 任何请求收到 401 时，自动用 refresh cookie 换发新会话并重试一次（见 coreFetch），
//     对调用方无感；刷新失败才上抛 401。
//   - 组件里不直接写 fetch —— 否则后端改响应格式要改几十处。

export class ApiError extends Error {
  readonly code: string
  readonly status?: number

  constructor(code: string, message: string, status?: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

const BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'
const TIMEOUT_MS = 15_000
const UPLOAD_TIMEOUT_MS = 60_000

// access token 仅存内存：刷新页面即失效，需靠 refresh cookie 重新换取。
let authToken: string | null = null

export function setAuthToken(token: string | null): void {
  authToken = token
}

export function getAuthToken(): string | null {
  return authToken
}

// 刷新处理器由 auth store 注册：调用后返回新的 access token（内部已写入 cookie）。
// 这样 api 层不直接依赖 auth 模块，避免循环引用。
let refreshHandler: (() => Promise<string>) | null = null
export function setRefreshHandler(fn: (() => Promise<string>) | null): void {
  refreshHandler = fn
}

// 并发请求同时 401 时，只发一次刷新（后端每次刷新会轮换 token，并发刷新会互相让对方失效）。
let refreshing: Promise<string> | null = null

type Params = Record<string, string | number | boolean | undefined>

function buildUrl(path: string, params?: Params): string {
  const url = new URL(BASE + path, window.location.origin)
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') url.searchParams.set(k, String(v))
    }
  }
  return url.toString()
}

async function parse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let msg = `请求失败（HTTP ${res.status}）`
    let code = `http_${res.status}`
    try {
      const j = await res.json()
      if (j?.error) {
        code = j.error.code ?? code
        msg = j.error.message ?? msg
      }
    } catch {
      /* 响应体不是 JSON 时忽略，沿用默认文案 */
    }
    throw new ApiError(code, msg, res.status)
  }
  const json = await res.json()
  if (json?.error) {
    throw new ApiError(json.error.code ?? 'unknown', json.error.message ?? '请求失败')
  }
  return json.data as T
}

interface FetchOpts {
  method: string
  path: string
  params?: Params
  body?: BodyInit
  // null 表示不主动设置 Content-Type（如 multipart/form-data 让浏览器自己带 boundary）
  contentType?: string | null
  timeoutMs?: number
}

// 发起单次请求并返回解析后的响应；自带独立超时控制器。
async function fetchOnce<T>(opts: FetchOpts, token: string | null): Promise<T> {
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), opts.timeoutMs ?? TIMEOUT_MS)

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (opts.contentType !== null) {
    headers['Content-Type'] = opts.contentType ?? 'application/json'
  }
  if (token) headers['Authorization'] = `Bearer ${token}`

  try {
    const res = await fetch(buildUrl(opts.path, opts.params), {
      method: opts.method,
      signal: ctrl.signal,
      headers,
      body: opts.body,
      credentials: 'include',
    })
    return await parse<T>(res)
  } finally {
    clearTimeout(timer)
  }
}

// 所有请求的统一执行体：负责超时、401 自动续期与单次重试。
async function coreFetch<T>(opts: FetchOpts): Promise<T> {
  try {
    return await fetchOnce<T>(opts, authToken)
  } catch (err) {
    // 401 且非刷新请求本身，且已注册刷新处理器 → 换发新会话后重试一次
    if (
      err instanceof ApiError &&
      err.status === 401 &&
      opts.path !== '/admin/refresh' &&
      refreshHandler
    ) {
      try {
        // 并发 401 共用同一次刷新结果，避免互相让对方的 refresh token 失效
        if (!refreshing) {
          refreshing = (async () => {
            try {
              return await refreshHandler!()
            } finally {
              refreshing = null
            }
          })()
        }
        const newToken = await refreshing
        setAuthToken(newToken)
        return await fetchOnce<T>(opts, newToken)
      } catch {
        // 刷新失败：抛出原始的 401，让调用方（如 query 错误回调）感知登录失效
        throw err
      }
    }
    if (err instanceof ApiError) throw err
    throw new ApiError('network_error', '连不上后端，确认服务已启动且地址正确')
  }
}

export async function get<T>(path: string, params?: Params): Promise<T> {
  return coreFetch<T>({ method: 'GET', path, params })
}

export async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  params?: Params,
): Promise<T> {
  return coreFetch<T>({
    method,
    path,
    params,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

// 文件上传：multipart/form-data。auth header 与 cookie 与 request() 一致。
export async function upload<T>(
  path: string,
  file: File,
  fieldName = 'file',
): Promise<T> {
  const form = new FormData()
  form.append(fieldName, file)
  return coreFetch<T>({
    method: 'POST',
    path,
    body: form,
    contentType: null,
    timeoutMs: UPLOAD_TIMEOUT_MS,
  })
}

export const api = {
  get,
  request,
  upload,
  setAuthToken,
  getAuthToken,
  setRefreshHandler,
}
