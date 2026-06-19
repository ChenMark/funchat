package handler

import (
	"fmt"
	"net/http"

	"funchat/backend/internal/repository"
	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"
	"funchat/backend/pkg/sms"
	"funchat/backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关 handler
type AuthHandler struct {
	jwtManager *jwt.Manager
	repo       *repository.Repo
	smsClient  *sms.Client
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(jwtManager *jwt.Manager, repo *repository.Repo, smsClient *sms.Client) *AuthHandler {
	return &AuthHandler{jwtManager: jwtManager, repo: repo, smsClient: smsClient}
}

// SendCode 发送短信验证码
// POST /api/v1/auth/send-code
func (h *AuthHandler) SendCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入手机号")
		return
	}

	if !validator.Phone(req.Phone) {
		response.BadRequest(c, "请输入正确的手机号")
		return
	}

	code, err := h.repo.CreateVerificationCode(req.Phone)
	if err != nil {
		switch err.Error() {
		case "daily_limit":
			response.Error(c, http.StatusTooManyRequests, 40013, "今日发送次数已达上限")
		case "too_frequent":
			response.Error(c, http.StatusTooManyRequests, 40012, "发送过于频繁，请稍后再试")
		default:
			response.InternalError(c)
		}
		return
	}

	// 发送真实短信（未配置时自动降级为控制台打印）
	if h.smsClient != nil {
		if err := h.smsClient.SendCode(req.Phone, code); err != nil {
			fmt.Printf("[SMS] 发送失败: %v, 降级打印验证码: %s\n", err, code)
		}
	} else {
		fmt.Printf("[SMS] 开发模式: 验证码=%s → %s\n", req.Phone, code)
	}

	response.Success(c, gin.H{
		"message": "验证码已发送",
		"code":    code, // 开发环境返回
	})
}

// VerifyCode 校验验证码
// POST /api/v1/auth/verify-code
func (h *AuthHandler) VerifyCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	codeToken, isNew, err := h.repo.VerifyCode(req.Phone, req.Code)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 40010, "验证码校验失败")
		return
	}
	if codeToken == "" {
		response.Error(c, http.StatusBadRequest, 40011, "验证码错误或已过期")
		return
	}

	response.Success(c, gin.H{
		"code_token": codeToken,
		"is_new":     isNew,
	})
}

// Register 新用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Phone     string `json:"phone" binding:"required"`
		CodeToken string `json:"code_token" binding:"required"`
		Nickname  string `json:"nickname"`
		Avatar    string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.repo.VerifyCodeToken(req.Phone, req.CodeToken) {
		response.Error(c, http.StatusBadRequest, 40010, "验证码已过期，请重新获取")
		return
	}

	user, err := h.repo.CreateUser(req.Phone, req.Nickname, req.Avatar)
	if err != nil {
		existing, getErr := h.repo.GetUserByPhone(req.Phone)
		if getErr != nil {
			response.Error(c, http.StatusInternalServerError, 50000, "注册失败")
			return
		}
		user = existing
	}

	accessToken, _ := h.jwtManager.GenerateAccessToken(user.ID, req.Phone)
	refreshToken, _ := h.jwtManager.GenerateRefreshToken(user.ID, req.Phone)

	response.Created(c, gin.H{
		"user_id":       user.ID,
		"nickname":      user.Nickname,
		"phone":         req.Phone,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    7200,
	})
}

// Login 手机号登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Phone     string `json:"phone" binding:"required"`
		CodeToken string `json:"code_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if !h.repo.VerifyCodeToken(req.Phone, req.CodeToken) {
		response.Error(c, http.StatusBadRequest, 40010, "验证码已过期，请重新获取")
		return
	}

	user, err := h.repo.GetUserByPhone(req.Phone)
	if err != nil {
		response.Error(c, http.StatusNotFound, 40401, "用户不存在，请先注册")
		return
	}

	accessToken, _ := h.jwtManager.GenerateAccessToken(user.ID, req.Phone)
	refreshToken, _ := h.jwtManager.GenerateRefreshToken(user.ID, req.Phone)

	response.Success(c, gin.H{
		"user_id":       user.ID,
		"phone":         req.Phone,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    7200,
	})
}

// RefreshToken 刷新 Token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	claims, err := h.jwtManager.ParseToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "Refresh Token 无效或已过期")
		return
	}

	accessToken, _ := h.jwtManager.GenerateAccessToken(claims.UserID, claims.Phone)
	refreshToken, _ := h.jwtManager.GenerateRefreshToken(claims.UserID, claims.Phone)

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    7200,
	})
}
