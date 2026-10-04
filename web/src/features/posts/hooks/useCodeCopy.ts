import { useEffect, type RefObject } from 'react'

// 给文章里的每个代码块挂一个复制按钮。
//
// 后端的渲染产物是静态 HTML，前端用 DOM 注入按钮而不是再解析一次 ——
// 正文已经是消毒过的 HTML，任何二次拼装都要小心不要破坏它。
export function useCodeCopy(ref: RefObject<HTMLElement | null>, deps: unknown[] = []) {
  useEffect(() => {
    const root = ref.current
    if (!root) return

    const cleanups: Array<() => void> = []
    root.querySelectorAll<HTMLElement>('.code-block').forEach((block) => {
      const pre = block.querySelector('pre')
      if (!pre) return

      const btn = document.createElement('button')
      btn.type = 'button'
      btn.className = 'copy-btn'
      btn.textContent = '复制'

      const onClick = async () => {
        try {
          await navigator.clipboard.writeText(pre.innerText)
          btn.textContent = '已复制'
          btn.dataset.state = 'ok'
        } catch {
          btn.textContent = '复制失败'
          btn.dataset.state = 'fail'
        }
        window.setTimeout(() => {
          btn.textContent = '复制'
          delete btn.dataset.state
        }, 1600)
      }

      btn.addEventListener('click', onClick)
      block.appendChild(btn)

      cleanups.push(() => {
        btn.removeEventListener('click', onClick)
        btn.remove()
      })
    })

    return () => cleanups.forEach((c) => c())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ref, ...deps])
}
