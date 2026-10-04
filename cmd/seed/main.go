// Command seed 把 testdata/posts/*.md 导入数据库，用于本地开发与渲染管线验证。
//
// 为什么要有它：M2 才会有后台编辑器，而 M1 要验证「文章能不能正确渲染出来」，
// 必须有真实数据。这批 Markdown 同时也充当渲染管线的测试样本。
//
// 行为是**幂等**的：按 slug 判重，已存在则更新内容与标签。可以反复跑。
//
// 用法：make seed
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"personal-blog/internal/config"
	"personal-blog/internal/pkg/markdown"
	"personal-blog/internal/pkg/slug"
	"personal-blog/internal/repository"
)

// statusPublished 对应 posts.status = 2
const statusPublished = 2

type meta struct {
	Title       string
	Slug        string
	Tags        []string
	PublishedAt time.Time
}

func main() {
	dir := flag.String("dir", "testdata/posts", "Markdown 文章目录")
	envFile := flag.String("env", ".env", "环境文件路径")
	dryRun := flag.Bool("dry-run", false, "只渲染并打印结果，不连数据库")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	renderer := markdown.New()

	files, err := filepath.Glob(filepath.Join(*dir, "*.md"))
	if err != nil {
		fatal("扫描目录失败: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		fatal("%s 下没有找到 .md 文件", *dir)
	}

	// dry-run 用来在没有数据库时检查渲染效果与元数据解析是否正确
	if *dryRun {
		for _, path := range files {
			raw, err := os.ReadFile(path)
			if err != nil {
				fatal("读取文件失败: %v", err)
			}
			m, body := parseFrontMatter(string(raw))
			if m.Slug == "" {
				m.Slug = slug.Make(m.Title)
			}
			res, err := renderer.Render(body)
			if err != nil {
				fatal("渲染 %s 失败: %v", path, err)
			}

			fmt.Printf("\n=== %s ===\n", filepath.Base(path))
			fmt.Printf("title       : %s\n", m.Title)
			fmt.Printf("slug        : %s\n", m.Slug)
			fmt.Printf("tags        : %v\n", m.Tags)
			fmt.Printf("published_at: %s\n", m.PublishedAt.Format("2006-01-02"))
			fmt.Printf("正文 %d 字符 / HTML %d 字符 / 目录 %d 项 / 阅读约 %d 分钟\n",
				len([]rune(body)), len(res.HTML), len(res.TOC), readingMinutes(res.PlainText))
			for _, t := range res.TOC {
				fmt.Printf("  %*s- %s (#%s)\n", (t.Level-1)*2, "", t.Text, t.ID)
			}
			fmt.Printf("摘要        : %s\n", markdown.Excerpt(res.PlainText, 200))
		}
		return
	}

	cfg, err := config.Load(*envFile)
	if err != nil {
		fatal("加载配置失败: %v", err)
	}

	db, err := repository.Open(cfg)
	if err != nil {
		fatal("连接数据库失败: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	q := repository.New(db)

	authorID, err := ensureAuthor(ctx, q, log)
	if err != nil {
		fatal("准备作者记录失败: %v", err)
	}

	// 真正入库时才需要知道本站域名（外链判断依赖它）
	renderer = markdown.New(markdown.WithSiteHost(hostOf(cfg.BaseURL)))

	ok, fail := 0, 0
	for _, path := range files {
		if err := seedFile(ctx, q, renderer, path, authorID); err != nil {
			log.Error("导入失败", slog.String("file", path), slog.String("err", err.Error()))
			fail++
			continue
		}
		log.Info("已导入", slog.String("file", filepath.Base(path)))
		ok++
	}

	log.Info("导入完成", slog.Int("成功", ok), slog.Int("失败", fail))
	if fail > 0 {
		os.Exit(1)
	}
}

func seedFile(ctx context.Context, q *repository.Queries, rd *markdown.Renderer, path string, authorID uint64) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件: %w", err)
	}

	m, body := parseFrontMatter(string(raw))
	if m.Title == "" {
		m.Title = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	if m.Slug == "" {
		m.Slug = slug.Make(m.Title)
	}
	if m.PublishedAt.IsZero() {
		m.PublishedAt = time.Now().UTC()
	}

	res, err := rd.Render(body)
	if err != nil {
		return fmt.Errorf("渲染 Markdown: %w", err)
	}

	tocJSON, err := json.Marshal(res.TOC)
	if err != nil {
		return fmt.Errorf("序列化目录: %w", err)
	}

	now := time.Now().UTC()
	summary := markdown.Excerpt(res.PlainText, 200)
	reading := readingMinutes(res.PlainText)

	existing, err := q.GetPostBySlugAny(ctx, m.Slug)
	switch {
	case err == nil:
		if err := q.UpdatePostByID(ctx, repository.UpdatePostByIDParams{
			Title:          m.Title,
			Summary:        summary,
			ContentMd:      body,
			ContentHtml:    res.HTML,
			TocJson:        nullString(string(tocJSON)),
			CoverUrl:       "",
			ReadingTimeMin: uint32(reading),
			PublishedAt:    nullTime(m.PublishedAt),
			UpdatedAt:      now,
			ID:             existing.ID,
		}); err != nil {
			return fmt.Errorf("更新文章: %w", err)
		}
		return relinkTags(ctx, q, existing.ID, m.Tags)

	case isNoRows(err):
		publicID := ulid.Make().String()
		id, err := q.CreatePost(ctx, repository.CreatePostParams{
			PublicID:       publicID,
			Slug:           m.Slug,
			Title:          m.Title,
			Summary:        summary,
			ContentMd:      body,
			ContentHtml:    res.HTML,
			TocJson:        nullString(string(tocJSON)),
			CoverUrl:       "",
			Status:         statusPublished,
			CommentEnabled: 1,
			ReadingTimeMin: uint32(reading),
			PublishedAt:    nullTime(m.PublishedAt),
			CreatedAt:      now,
			UpdatedAt:      now,
			AuthorID:       authorID,
		})
		if err != nil {
			return fmt.Errorf("创建文章: %w", err)
		}
		return relinkTags(ctx, q, uint64(id), m.Tags)

	default:
		return fmt.Errorf("查询文章: %w", err)
	}
}

// relinkTags 先清空再重建关联，保证「改了 front-matter 里的标签」能真正生效。
func relinkTags(ctx context.Context, q *repository.Queries, postID uint64, names []string) error {
	if err := q.DeletePostTags(ctx, postID); err != nil {
		return fmt.Errorf("清空标签关联: %w", err)
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tagSlug := slug.Make(name)

		tag, err := q.GetTagBySlug(ctx, tagSlug)
		if err != nil {
			if !isNoRows(err) {
				return fmt.Errorf("查询标签: %w", err)
			}
			id, err := q.CreateTag(ctx, repository.CreateTagParams{Slug: tagSlug, Name: name})
			if err != nil {
				return fmt.Errorf("创建标签: %w", err)
			}
			tag.ID = uint64(id)
		}

		if err := q.LinkPostTag(ctx, repository.LinkPostTagParams{PostID: postID, TagID: tag.ID}); err != nil {
			return fmt.Errorf("关联标签: %w", err)
		}
	}
	return nil
}

// ensureAuthor 保证 posts.author_id 有值可填。
//
// 这个账号是**占位**的：password_hash 故意不以 argon2 开头，
// M2 的密码校验会直接拒绝它，所以不存在「默认密码」的风险。
func ensureAuthor(ctx context.Context, q *repository.Queries, log *slog.Logger) (uint64, error) {
	total, err := q.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("统计用户: %w", err)
	}
	if total > 0 {
		// 已有用户时直接取第一个的 id —— seed 只是本地工具，不值得为它加复杂逻辑
		row, err := q.GetUserByEmail(ctx, seedAuthorEmail)
		if err == nil {
			return row.ID, nil
		}
		if !isNoRows(err) {
			return 0, fmt.Errorf("查询作者: %w", err)
		}
	}

	id, err := q.CreateUser(ctx, repository.CreateUserParams{
		PublicID:     ulid.Make().String(),
		Email:        seedAuthorEmail,
		PasswordHash: "!placeholder-not-loginable",
		DisplayName:  "博主",
		CanUpload:    1,
	})
	if err != nil {
		return 0, fmt.Errorf("创建占位作者: %w", err)
	}
	log.Info("已创建占位作者（不可登录）", slog.String("email", seedAuthorEmail))
	return uint64(id), nil
}

const seedAuthorEmail = "author@seed.local"

// parseFrontMatter 解析文件开头的 YAML front-matter。
//
// 只支持本项目用到的四个字段（title / slug / tags / published_at），
// 不引入完整的 YAML 解析库 —— 为一小段元数据拖进一个依赖不划算。
func parseFrontMatter(content string) (meta, string) {
	var m meta

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return m, content
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return m, content
	}

	for _, line := range lines[1:end] {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "title":
			m.Title = value
		case "slug":
			m.Slug = value
		case "tags":
			// 形如 [Go, 并发]
			value = strings.TrimPrefix(value, "[")
			value = strings.TrimSuffix(value, "]")
			for _, part := range strings.Split(value, ",") {
				if t := strings.TrimSpace(part); t != "" {
					m.Tags = append(m.Tags, t)
				}
			}
		case "published_at":
			for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
				if t, err := time.Parse(layout, value); err == nil {
					m.PublishedAt = t.UTC()
					break
				}
			}
		}
	}

	return m, strings.Join(lines[end+1:], "\n")
}

// readingMinutes 粗估阅读时长。
//
// 中文按 400 字/分钟、代码与英文按 200 词/分钟过于精细，
// 这里统一按 500 字符/分钟估计 —— 误差在 ±1 分钟内，对这个场景足够。
func readingMinutes(plain string) int {
	n := len([]rune(strings.TrimSpace(plain)))
	m := n / 500
	if n%500 > 0 {
		m++
	}
	if m < 1 {
		m = 1
	}
	return m
}

func hostOf(baseURL string) string {
	if i := strings.Index(baseURL, "://"); i >= 0 {
		rest := baseURL[i+3:]
		if j := strings.IndexAny(rest, "/:"); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return ""
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.UTC(), Valid: true}
}

func isNoRows(err error) bool { return err == sql.ErrNoRows || strings.Contains(err.Error(), "no rows") }

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seed: "+format+"\n", args...)
	os.Exit(1)
}
