// Package media 负责图片落盘与处理。
//
// 设计取舍（这些约束必须在写代码时就成立，事后补会很别扭）：
//
//  1. **文件名一律服务端生成** `<ULID>.<ext>`。一次性解决三件事：
//     路径穿越（`../../etc/passwd`）、中文名乱码、同名覆盖。
//     客户端传来的文件名一律不采信，只用于展示。
//  2. **MIME 读文件头判断**，不信任扩展名也不信任客户端传的 Content-Type ——
//     把 `.php` 改名成 `.png` 是最经典的上传绕过手法。
//  3. **禁 SVG**：SVG 能内嵌脚本，是 XSS 载体，本站不接受。
//  4. **不接对象存储抽象层**。用户明确图片存本地，为一个"将来可能"的需求
//     预留接口是过度设计；真要换时改动面就在这个包。
package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // 注册 webp 解码器，否则 webp 上传会被当成未知格式拒绝
)

// 允许的 MIME。故意不含 image/svg+xml —— SVG 能内嵌 <script>，是 XSS 载体。
var allowedMIME = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// 图像处理参数。
const (
	MaxWidth    = 1600 // 超过则等比缩放
	JPEGQuality = 82   // 肉眼无差别但体积明显更小的档位
)

// ErrUnsupportedType 表示文件类型不在白名单内。
var ErrUnsupportedType = errors.New("不支持的文件类型")

// Store 管理磁盘上的媒体文件。
type Store struct {
	root      string // 磁盘根目录（MEDIA_DIR）
	urlPrefix string // URL 前缀（MEDIA_URL_PREFIX）
}

// Result 是保存成功的产物，供 service 层写库。
type Result struct {
	Filename  string // 磁盘文件名，不含目录
	RelPath   string // 相对 root 的路径，如 2026/09/01JXYZ....jpg
	URL       string // 对外可访问的 URL
	MimeType  string
	SizeBytes int64
	Width     int
	Height    int
}

// NewStore 构造存储器，并确保根目录存在。
func NewStore(root, urlPrefix string) (*Store, error) {
	if root == "" {
		return nil, errors.New("MEDIA_DIR 不能为空")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建媒体目录 %s: %w", root, err)
	}
	return &Store{root: root, urlPrefix: strings.TrimSuffix(urlPrefix, "/")}, nil
}

// Save 处理并落盘。src 是完整文件内容，调用方必须先用
// http.MaxBytesReader 限制大小，避免把超大文件读进内存。
func (s *Store) Save(src []byte) (*Result, error) {
	// 1. 读文件头判断真实类型
	mime := Detect(src)
	ext, ok := allowedMIME[mime]
	if !ok {
		return nil, fmt.Errorf("%w: %s（只接受 JPEG / PNG / GIF / WebP）", ErrUnsupportedType, mime)
	}

	// 2. 处理（压缩 + 限宽）。GIF 例外：重编码会丢动画，原样保留。
	out, outMIME, w, h, err := process(src, mime)
	if err != nil {
		return nil, err
	}
	if mime == "image/gif" {
		gw, gh := gifSize(src)
		out, outMIME, w, h = src, mime, gw, gh
	}
	if outMIME != mime {
		ext = allowedMIME[outMIME]
	}

	// 3. 服务端生成文件名，按 年月 分目录（避免单目录文件数过多）
	now := time.Now().UTC()
	filename := ulid.Make().String() + "." + ext
	rel := filepath.Join(now.Format("2006"), now.Format("01"), filename)

	abs := filepath.Join(s.root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, fmt.Errorf("创建子目录: %w", err)
	}
	// 0o644：容器里以非 root 运行，只需给 web server 读权限，绝不给可执行位
	if err := os.WriteFile(abs, out, 0o644); err != nil {
		return nil, fmt.Errorf("写入文件: %w", err)
	}

	return &Result{
		Filename:  filename,
		RelPath:   rel,
		URL:       s.urlPrefix + "/" + filepath.ToSlash(rel),
		MimeType:  outMIME,
		SizeBytes: int64(len(out)),
		Width:     w,
		Height:    h,
	}, nil
}

// Delete 删除文件。传 relPath（库里存的路径），不是绝对路径。
func (s *Store) Delete(relPath string) error {
	rel, err := safeRel(relPath)
	if err != nil {
		return err
	}
	abs := filepath.Join(s.root, rel)
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除文件: %w", err)
	}
	return nil
}

// UsedBytes 统计已占用磁盘空间，用于配额校验。
func (s *Store) UsedBytes() (int64, error) {
	var total int64
	err := filepath.WalkDir(s.root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil // 单个文件 stat 失败不值得让整个上传失败
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("统计媒体占用: %w", err)
	}
	return total, nil
}

// Detect 读取文件头判断 MIME。
//
// 用标准库的 sniff 而不是扩展名：扩展名由客户端控制，不可信。
func Detect(src []byte) string {
	return http.DetectContentType(src)
}

// --- 内部 ---

// process 解码 → 限宽 → 重编码。
func process(src []byte, mime string) ([]byte, string, int, int, error) {
	img, format, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("图片解码失败（文件可能已损坏）: %w", err)
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	if w > MaxWidth {
		nh := h * MaxWidth / w
		dst := image.NewRGBA(image.Rect(0, 0, MaxWidth, nh))
		// CatmullRom 比双线性慢，但对文字截图这类高频内容明显更清晰
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img, w, h = dst, MaxWidth, nh
	}

	// 有透明通道的图转 JPEG 会把透明区变成黑块，只能留 PNG
	if opaque(img) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
			return nil, "", 0, 0, fmt.Errorf("JPEG 编码: %w", err)
		}
		// 原图就是 JPEG 且重编码后反而更大时，保留原样 —— 避免"压缩"越压越大
		if format == "jpeg" && int64(buf.Len()) >= int64(len(src)) {
			return src, mime, w, h, nil
		}
		return buf.Bytes(), "image/jpeg", w, h, nil
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, "", 0, 0, fmt.Errorf("PNG 编码: %w", err)
	}
	if format == "png" && int64(buf.Len()) >= int64(len(src)) {
		return src, mime, w, h, nil
	}
	return buf.Bytes(), "image/png", w, h, nil
}

// opaque 判断图像是否完全不透明。
func opaque(img image.Image) bool {
	o, ok := img.(interface{ Opaque() bool })
	return ok && o.Opaque()
}

// gifSize 取 GIF 的首帧尺寸（不解码全部帧，够用了）。
func gifSize(src []byte) (int, int) {
	if cfg, err := gif.DecodeConfig(bytes.NewReader(src)); err == nil {
		return cfg.Width, cfg.Height
	}
	return 0, 0
}

// safeRel 校验相对路径，杜绝路径穿越。
//
// 文件名虽然由服务端生成，但 Delete 的入参来自库记录，
// 库被注入时这条是最后一道防线。
func safeRel(rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") || strings.Contains(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("非法的文件路径: %q", rel)
	}
	return clean, nil
}
