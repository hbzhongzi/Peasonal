package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"personal-blog/internal/domain"
	"personal-blog/internal/pkg/apierr"
	"personal-blog/internal/pkg/auth"
	"personal-blog/internal/repository"
)

// emailRe：本地部分@域名.后缀。HTML5 的 type=email 连 "12@123" 都放行，
// 服务端必须自己把住关 —— 无「.后缀」的地址收进来，昵称默认值和找回密码都会跟着遭殃。
var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	accessIssuer    = "personal-blog"
)

// AuthService 负责登录、令牌刷新、登出与首个管理员引导。
//
// 返回结构里不含 cookie：refresh token 是传输层概念，
// 由 handler 负责把它写进 HttpOnly cookie（见传输层）。
type AuthService struct {
	q          *repository.Queries
	tokens     *auth.Manager
	refreshTTL time.Duration
}

func NewAuthService(q *repository.Queries, jwtSecret string) *AuthService {
	return &AuthService{
		q:          q,
		tokens:     auth.NewManager(jwtSecret, accessIssuer, accessTokenTTL),
		refreshTTL: refreshTokenTTL,
	}
}

// LoginResult 是登录/刷新成功的产物。
type LoginResult struct {
	AccessToken     string
	AccessExpiresAt time.Time
	// RefreshToken 是明文，handler 负责写进 HttpOnly cookie。
	RefreshToken string
	User        *domain.AdminUser
}

// Login 校验邮箱密码，成功则签发一对新令牌。
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, apierr.InvalidArgument("邮箱和密码不能为空")
	}
	user, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.Unauthorized("邮箱或密码错误")
		}
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	ok, err := auth.VerifyPassword(user.PasswordHash, password)
	if err != nil {
		// 哈希格式损坏等，记日志但不外泄细节
		slog.Error("密码哈希校验异常", "user_id", user.ID, "err", err)
		return nil, apierr.Unauthorized("邮箱或密码错误")
	}
	if !ok {
		return nil, apierr.Unauthorized("邮箱或密码错误")
	}
	return s.issueSession(ctx, *adminUserFromRow(user.ID, user.PublicID, user.Email, user.DisplayName, user.CanUpload, user.CanPublish, user.IsAdmin, user.Bio, user.AvatarUrl, user.Github))
}

// Refresh 用 refresh token 换一个新会话（同时轮换 refresh token）。
func (s *AuthService) Refresh(ctx context.Context, refreshPlain string) (*LoginResult, error) {
	if refreshPlain == "" {
		return nil, apierr.Unauthorized("缺少刷新令牌")
	}
	hash := auth.HashToken(refreshPlain)
	rt, err := s.q.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.Unauthorized("刷新令牌无效")
		}
		return nil, fmt.Errorf("查询刷新令牌: %w", err)
	}
	if rt.RevokedAt.Valid {
		return nil, apierr.Unauthorized("刷新令牌已失效")
	}
	if rt.ExpiresAt.Before(time.Now().UTC()) {
		return nil, apierr.Unauthorized("刷新令牌已过期")
	}
	// 轮换：先撤销旧的，再签发全新会话，降低重放窗口
	if err := s.q.RevokeRefreshToken(ctx, repository.RevokeRefreshTokenParams{
		RevokedAt: nullTime(time.Now().UTC()),
		TokenHash: hash,
	}); err != nil {
		return nil, fmt.Errorf("撤销旧刷新令牌: %w", err)
	}
	user, err := s.q.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	return s.issueSession(ctx, *adminUserFromRow(user.ID, user.PublicID, user.Email, user.DisplayName, user.CanUpload, user.CanPublish, user.IsAdmin, user.Bio, user.AvatarUrl, user.Github))
}

// Logout 撤销当前 refresh token（其余设备的会话不受影响）。
func (s *AuthService) Logout(ctx context.Context, refreshPlain string) error {
	if refreshPlain == "" {
		return nil
	}
	if err := s.q.RevokeRefreshToken(ctx, repository.RevokeRefreshTokenParams{
		RevokedAt: nullTime(time.Now().UTC()),
		TokenHash: auth.HashToken(refreshPlain),
	}); err != nil {
		return fmt.Errorf("撤销刷新令牌: %w", err)
	}
	return nil
}

// Me 返回当前登录用户的信息。
func (s *AuthService) Me(ctx context.Context, userID uint64) (*domain.AdminUser, error) {
	user, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("用户不存在")
		}
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	return adminUserFromRow(user.ID, user.PublicID, user.Email, user.DisplayName, user.CanUpload, user.CanPublish, user.IsAdmin, user.Bio, user.AvatarUrl, user.Github), nil
}

// MyStats 返回当前登录用户的「我的视角」发布统计（已发布口径）。
// 首页 Hero 与侧边栏个人卡片在登录态下展示这些数字——
// 换个人登录，首页身份区就该换成那个人的文章数，而不是永远站长的。
func (s *AuthService) MyStats(ctx context.Context, userID uint64) (*domain.MyStats, error) {
	row, err := s.q.GetAuthorPublishStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询我的发布统计: %w", err)
	}
	return &domain.MyStats{
		PostTotal:  row.PostTotal,
		TagTotal:   row.TagTotal,
		MonthTotal: row.MonthTotal,
	}, nil
}

// Register 公开注册：创建普通用户（可评论/上传，但不可发帖），注册成功后直接签发会话实现「注册即登录」。
func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*LoginResult, error) {
	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)
	if email == "" || password == "" {
		return nil, apierr.InvalidArgument("邮箱和密码不能为空")
	}
	if !emailRe.MatchString(email) || len(email) > 254 {
		return nil, apierr.InvalidArgument("邮箱格式不正确")
	}
	if len(password) < 8 {
		return nil, apierr.InvalidArgument("密码至少 8 位")
	}
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}

	if _, err := s.q.GetUserByEmail(ctx, email); err == nil {
		return nil, apierr.Conflict("该邮箱已注册")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("查询邮箱是否重复: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("哈希密码: %w", err)
	}
	id, err := s.q.CreateUser(ctx, repository.CreateUserParams{
		PublicID:     newULID(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		CanUpload:    0,    // 上传为管理员授予的特权：默认关闭，由管理员在用户管理页开启
		CanPublish:   1,    // 注册用户默认即为作者，可在「我的博客」写/发文章（见 M2 补充八）
		IsAdmin:      0,    // 公开注册永远不是站点管理员
	})
	if err != nil {
		return nil, fmt.Errorf("创建用户: %w", err)
	}
	user, err := s.q.GetUserByID(ctx, uint64(id))
	if err != nil {
		return nil, fmt.Errorf("读取新用户: %w", err)
	}
	return s.issueSession(ctx, *adminUserFromRow(user.ID, user.PublicID, user.Email, user.DisplayName, user.CanUpload, user.CanPublish, user.IsAdmin, user.Bio, user.AvatarUrl, user.Github))
}

// ListUsers 返回全部用户（管理员用），不含密码哈希。
func (s *AuthService) ListUsers(ctx context.Context) ([]domain.AdminUser, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("列出用户: %w", err)
	}
	out := make([]domain.AdminUser, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AdminUser{
			ID:          r.ID,
			PublicID:    r.PublicID,
			Email:       r.Email,
			DisplayName: r.DisplayName,
			CanUpload:   r.CanUpload != 0,
			CanPublish:  r.CanPublish != 0,
			IsAdmin:     r.IsAdmin != 0,
		})
	}
	return out, nil
}

// UpdatePermissions 由站点管理员开启/关闭某用户的发帖与上传权限，并可升降其管理员身份。
func (s *AuthService) UpdatePermissions(ctx context.Context, publicID string, canUpload, canPublish, canAdmin bool) (*domain.AdminUser, error) {
	if err := s.q.UpdateUserPermissions(ctx, repository.UpdateUserPermissionsParams{
		CanUpload:  boolToInt8(canUpload),
		CanPublish: boolToInt8(canPublish),
		IsAdmin:    boolToInt8(canAdmin),
		PublicID:   publicID,
	}); err != nil {
		return nil, fmt.Errorf("更新用户权限: %w", err)
	}
	user, err := s.q.GetUserByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("读取更新后的用户: %w", err)
	}
	return adminUserFromRow(user.ID, user.PublicID, user.Email, user.DisplayName, user.CanUpload, user.CanPublish, user.IsAdmin, user.Bio, user.AvatarUrl, user.Github), nil
}

// CreateFirstAdmin 在库里还没有任何管理员时，用给定凭据创建一个。
// 已由 cmd/boot 在启动时调用，保证开发/首次部署有可用账号。
func (s *AuthService) CreateFirstAdmin(ctx context.Context, email, password, displayName string) (*domain.AdminUser, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, apierr.InvalidArgument("管理员邮箱和密码不能为空")
	}
	if n, err := s.q.CountUsers(ctx); err != nil {
		return nil, fmt.Errorf("统计用户数: %w", err)
	} else if n > 0 {
		return nil, apierr.Conflict("已存在管理员，无法重复引导")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("哈希密码: %w", err)
	}
	id, err := s.q.CreateUser(ctx, repository.CreateUserParams{
		PublicID:     newULID(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		CanUpload:    1,
		CanPublish:   1, // 首位管理员默认拥有发布权限
		IsAdmin:      1, // 首位引导出来的管理员同时也是站点管理员
	})
	if err != nil {
		return nil, fmt.Errorf("创建管理员: %w", err)
	}
	return &domain.AdminUser{
		ID:          uint64(id),
		Email:       email,
		DisplayName: displayName,
		CanUpload:   true,
		CanPublish:  true,
		IsAdmin:     true,
	}, nil
}

// ProfileInput 是本人修改公开身份的输入。
type ProfileInput struct {
	DisplayName string
	Bio         string
	AvatarURL   string
	Github      string
}

// UpdateMyProfile 由本人修改自己的公开身份（昵称/简介/头像/GitHub）。
//
// userID 只能取自 access token，不接受路径参数 —— 否则「改资料」就变成「改别人的资料」。
// 权限位（can_upload/can_publish/is_admin）不在这里改，那是管理员的事（UpdatePermissions）。
func (s *AuthService) UpdateMyProfile(ctx context.Context, userID uint64, in ProfileInput) (*domain.AdminUser, error) {
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		return nil, apierr.InvalidArgument("昵称不能为空")
	}
	if len([]rune(displayName)) > 64 {
		return nil, apierr.InvalidArgument("昵称最长 64 个字符")
	}
	bio := strings.TrimSpace(in.Bio)
	if len([]rune(bio)) > 512 {
		return nil, apierr.InvalidArgument("简介最长 512 个字符")
	}
	avatar := strings.TrimSpace(in.AvatarURL)
	if avatar != "" && !strings.HasPrefix(avatar, "/") && !strings.HasPrefix(avatar, "https://") {
		// 只接受站内路径与 https 外链：拦掉 javascript: / data: 这类会被当 URL 执行的输入
		return nil, apierr.InvalidArgument("头像地址必须是站内路径（/uploads/...）或 https 链接")
	}
	github := strings.TrimSpace(in.Github)
	if github != "" && !strings.HasPrefix(github, "https://") && !strings.HasPrefix(github, "http://") {
		return nil, apierr.InvalidArgument("GitHub 链接必须以 http:// 或 https:// 开头")
	}

	if err := s.q.UpdateMyProfile(ctx, repository.UpdateMyProfileParams{
		DisplayName: displayName,
		Bio:         bio,
		AvatarUrl:   avatar,
		Github:      github,
		ID:          userID,
	}); err != nil {
		return nil, fmt.Errorf("更新个人资料: %w", err)
	}
	return s.Me(ctx, userID)
}

// SiteProfile 返回站长的公开资料（取 id 最小的管理员）。
//
// 这个接口是匿名的，前台首页/关于页都靠它渲染博主卡片。
// 只回愿意公开的四项，不含邮箱 —— 邮箱一旦进了匿名接口，就是爬虫的收获。
func (s *AuthService) SiteProfile(ctx context.Context) (*domain.SiteProfile, error) {
	row, err := s.q.GetFirstAdminProfile(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierr.NotFound("站点资料尚未配置")
		}
		return nil, fmt.Errorf("查询站长资料: %w", err)
	}
	return &domain.SiteProfile{
		PublicID:    row.PublicID,
		DisplayName: row.DisplayName,
		Bio:         row.Bio,
		AvatarURL:   row.AvatarUrl,
		Github:      row.Github,
	}, nil
}

// issueSession 生成 access + refresh 令牌对，并把 refresh 哈希入库。
func (s *AuthService) issueSession(ctx context.Context, user domain.AdminUser) (*LoginResult, error) {
	access, exp, err := s.tokens.IssueAccessToken(user.ID, user.Email, user.CanUpload, user.CanPublish, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("签发 access token: %w", err)
	}
	refresh, err := auth.NewRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("生成 refresh token: %w", err)
	}
	now := time.Now().UTC()
	if err := s.q.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: auth.HashToken(refresh),
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("写入刷新令牌: %w", err)
	}
	return &LoginResult{
		AccessToken:     access,
		AccessExpiresAt: exp,
		RefreshToken:    refresh,
		User:            &user,
	}, nil
}

// adminUserFromRow 把仓储层用户行转换为对外的安全视图（不含密码哈希）。
// sqlc 在 omit_unused_structs 下为每个查询生成独立的 Row 类型，这里用字段逐个取，
// 各 Row 类型（GetUserByEmail/ID/PublicID/ListUsers）字段一致，复用同一转换。
func adminUserFromRow(id uint64, publicID, email, displayName string, canUpload, canPublish, isAdmin int8, bio, avatarURL, github string) *domain.AdminUser {
	return &domain.AdminUser{
		ID:          id,
		PublicID:    publicID,
		Email:       email,
		DisplayName: displayName,
		CanUpload:   canUpload != 0,
		CanPublish:  canPublish != 0,
		IsAdmin:     isAdmin != 0,
		Bio:         bio,
		AvatarURL:   avatarURL,
		Github:      github,
	}
}
