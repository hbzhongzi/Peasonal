---
title: Go 并发：errgroup 与有界并发
slug: go-concurrency-errgroup
tags: [Go, 并发]
published_at: 2026-09-08
---

# Go 并发：errgroup 与有界并发

Go 的并发原语足够简单，简单到很容易写出「能跑但有隐患」的代码。这篇记录两个高频场景的正确写法。

## 场景一：一组任务，任一失败就整体取消

最朴素的写法是用 `WaitGroup` 加一个 `error` 变量，但那个变量需要加锁，而且**取消传播**要自己做。`errgroup` 把这些都处理了：

```go
func fetchAll(ctx context.Context, urls []string) ([][]byte, error) {
	g, ctx := errgroup.WithContext(ctx)

	results := make([][]byte, len(urls))
	for i, u := range urls {
		i, u := i, u // Go 1.22 之前必须重新绑定
		g.Go(func() error {
			body, err := fetch(ctx, u)
			if err != nil {
				return fmt.Errorf("抓取 %s 失败: %w", u, err)
			}
			results[i] = body
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
```

关键点：

1. `errgroup.WithContext` 返回的 ctx，会在**任意一个** goroutine 返回错误时被取消
2. `g.Wait()` 只返回**第一个**错误，后续错误会被丢弃 —— 需要收集全部错误就得自己加切片
3. 每个 goroutine 写 `results` 的不同下标，不需要加锁

## 场景二：限制并发数

`g.Go` 会立刻启动 goroutine。如果 `urls` 有一万条，就是一万个并发请求 —— 对端的数据库连接池会先撑不住。

用带缓冲的 channel 做信号量：

```go
func fetchAllBounded(ctx context.Context, urls []string, limit int) error {
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, limit)

	for _, u := range urls {
		u := u
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return ctx.Err()
			}

			_, err := fetch(ctx, u)
			return err
		})
	}

	return g.Wait()
}
```

> `select` 里同时监听信号量与 `ctx.Done()` 是必须的。只监听信号量的话，一旦 ctx 被取消，这些 goroutine 会全部卡在等待信号量上，`Wait()` 永远不返回。

## 一个容易漏的细节

`errgroup` 不会 recover panic。任意一个 goroutine panic，整个进程就崩了。

如果这个任务跑在服务里，得自己包一层：

```go
g.Go(func() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return doWork(ctx)
})
```

## 并发数该设多少

不要凭感觉。经验起点是 `min(CPU 核数, 目标服务的承受能力)`：

| 任务类型 | 起点 |
| --- | --- |
| CPU 密集 | `runtime.NumCPU()` |
| 本地 I/O | 2 × `NumCPU()` |
| 调外部 HTTP | 对方限流值的一半 |
| 查数据库 | 连接池上限的 70% |

最后一条很重要：并发数超过连接池上限时，多出来的 goroutine 只是在排队，还白占了一份内存。
