// Package auth 负责密码哈希与令牌签发。
//
// 安全约定（改这里之前先读懂为什么）：
//
//  1. 密码用 argon2id 而不是 bcrypt —— 参数（内存、迭代、并行度）可控，
//     能随机器性能调；bcrypt 的成本因子上限受实现限制。
//  2. access token 是 JWT，**只存前端内存**，不放 localStorage（防 XSS 窃取）。
//  3. refresh token 是随机串，不是 JWT —— 它需要能被服务端主动撤销，
//     而 JWT 一旦签发在过期前无法作废，除非查库（那就失去了 JWT 的意义）。
//     库里只存 refresh token 的哈希，库被读走也无法冒用。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

// --- 密码 ---

// argon2id 参数。64MB 内存是 OWASP 给个人站点的推荐下限，
// 4C4G 机器上单次哈希约 50ms，登录接口的频率下不构成压力。
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 2
	argonKeyLen  = 32
	saltLen      = 16
)

// ErrBadPasswordHash 表示库里存的哈希格式不对。
var ErrBadPasswordHash = errors.New("密码哈希格式非法")

// HashPassword 生成 argon2id 哈希串。
//
// 输出格式与 crypt 的通行表示一致（`$argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>`），
// 自带全部参数是为了将来调参数时，旧哈希仍能用自己记录的参数校验，不必强制所有人改密码。
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成盐: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword 校验密码。哈希格式非法时返回 ErrBadPasswordHash。
func VerifyPassword(hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	// "$argon2id$v=19$m=..,t=..,p=..$salt$key" 按 $ 切分后是 6 段（首段为空）
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrBadPasswordHash
	}

	var memory uint32
	var timeCost, threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, ErrBadPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrBadPasswordHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrBadPasswordHash
	}

	got := argon2.IDKey([]byte(password), salt, uint32(timeCost), memory, threads, uint32(len(want)))
	// 用恒定时间比较，避免通过响应耗时逐字节爆破
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// --- 令牌 ---

// Token 类型标记，防止把 refresh token 当 access token 用。
type tokenType string

const (
	TypeAccess  tokenType = "access"
	TypeRefresh tokenType = "refresh"
)

// Manager 签发与校验 access token。
type Manager struct {
	secret   []byte
	issuer   string
	lifetime time.Duration
}

// NewManager 构造令牌管理器。lifetime 是 access token 的有效期。
func NewManager(secret, issuer string, lifetime time.Duration) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, lifetime: lifetime}
}

// Claims 是 access token 的载荷。
//
// UserID 用内部自增 id：token 不外泄给第三方，没必要为它再做一层映射。
// CanUpload / CanPublish 一并写入，是为了让鉴权中间件不必每次都查库就能判断权限。
type Claims struct {
	jwt.RegisteredClaims
	Type        tokenType `json:"typ"`
	Email       string    `json:"email,omitempty"`
	CanUpload   bool      `json:"cu,omitempty"`
	CanPublish  bool      `json:"cp,omitempty"`
	IsAdmin     bool      `json:"ia,omitempty"`
}

// IssueAccessToken 签发 access token。
//
// canUpload / canPublish / isAdmin 写入载荷，供中间件判断权限（见上）。
func (m *Manager) IssueAccessToken(userID uint64, email string, canUpload, canPublish, isAdmin bool) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.lifetime)

	c := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprint(userID),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
		},
		Type:       TypeAccess,
		Email:      email,
		CanUpload:  canUpload,
		CanPublish: canPublish,
		IsAdmin:    isAdmin,
	}

	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("签发 access token: %w", err)
	}
	return s, exp, nil
}

// ParseAccessToken 校验并解析 access token。
func (m *Manager) ParseAccessToken(token string) (*Claims, error) {
	var c Claims
	if _, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer)); err != nil {
		return nil, fmt.Errorf("解析 access token: %w", err)
	}
	if c.Type != TypeAccess {
		return nil, errors.New("令牌类型不是 access")
	}
	return &c, nil
}

// UserID 从 claims 中取回用户 id。
func (c *Claims) UserID() (uint64, error) {
	var id uint64
	if _, err := fmt.Sscan(c.Subject, &id); err != nil {
		return 0, fmt.Errorf("令牌 subject 非法: %w", err)
	}
	return id, nil
}

// --- refresh token ---

// NewRefreshToken 生成一个新的 refresh token（随机串，非 JWT）。
//
// 32 字节随机 = 256 位熵，不可能被猜到。
func NewRefreshToken() (plain string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成 refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken 计算 refresh token 的 SHA-256 十六进制表示。
//
// 用 SHA-256 而不是 bcrypt：token 本身已是高熵随机串，不存在暴力破解空间，
// 慢哈希只会拖慢每次刷新。而且需要能按哈希值**精确查询**该行来撤销它。
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
