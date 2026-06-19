package handler

import (
	"fmt"
	"time"

	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ThirdPartyHandler 第三方登录 handler
type ThirdPartyHandler struct {
	jwtManager *jwt.Manager
}

// NewThirdPartyHandler 创建 ThirdPartyHandler
func NewThirdPartyHandler(jwtManager *jwt.Manager) *ThirdPartyHandler {
	return &ThirdPartyHandler{jwtManager: jwtManager}
}

// WechatLogin 微信登录
// POST /api/v1/auth/wechat/login
func (h *ThirdPartyHandler) WechatLogin(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少微信授权 code")
		return
	}

	// TODO: 调用微信 API 用 code 换取 access_token + UnionID
	// wechatResp := wechat.GetAccessToken(req.Code)
	// unionID := wechatResp.UnionID

	// 模拟微信返回
	unionID := fmt.Sprintf("wx_%s", req.Code[:8])

	// TODO: 查数据库判断是否已有绑定
	phone := "" // 从 DB 查

	if phone != "" {
		// 已绑定 → 直接登录
		userID := fmt.Sprintf("u_%s", phone)
		accessToken, _ := h.jwtManager.GenerateAccessToken(userID, phone)
		refreshToken, _ := h.jwtManager.GenerateRefreshToken(userID, phone)
		response.Success(c, gin.H{
			"user_id":       userID,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"is_new":        false,
		})
		return
	}

	// 未绑定 → 返回 bind_token
	bindToken := fmt.Sprintf("bt_%s_%d", unionID, time.Now().UnixNano())
	response.Success(c, gin.H{
		"bind_token": bindToken,
		"union_id":   unionID,
		"is_new":     true,
		"need_bind":  true,
	})
}

// WechatBind 微信绑定手机号
// POST /api/v1/auth/wechat/bind
func (h *ThirdPartyHandler) WechatBind(c *gin.Context) {
	var req struct {
		BindToken string `json:"bind_token" binding:"required"`
		Phone     string `json:"phone" binding:"required"`
		CodeToken string `json:"code_token" binding:"required"`
		Nickname  string `json:"nickname"`
		Avatar    string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// TODO: 验证 code_token + 写入数据库

	userID := fmt.Sprintf("u_%s", req.Phone)
	accessToken, _ := h.jwtManager.GenerateAccessToken(userID, req.Phone)
	refreshToken, _ := h.jwtManager.GenerateRefreshToken(userID, req.Phone)

	response.Created(c, gin.H{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    7200,
	})
}

// AppleLogin Apple ID 登录
// POST /api/v1/auth/apple/login
func (h *ThirdPartyHandler) AppleLogin(c *gin.Context) {
	var req struct {
		IdentityToken string `json:"identity_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少 Apple identity_token")
		return
	}

	// TODO: 验证 identity_token (Apple 公钥验证)
	// appleUserID := apple.Verify(req.IdentityToken)
	appleUserID := fmt.Sprintf("apple_%s", req.IdentityToken[:8])

	// TODO: 查数据库判断是否已有绑定
	phone := ""

	if phone != "" {
		userID := fmt.Sprintf("u_%s", phone)
		accessToken, _ := h.jwtManager.GenerateAccessToken(userID, phone)
		refreshToken, _ := h.jwtManager.GenerateRefreshToken(userID, phone)
		response.Success(c, gin.H{
			"user_id":       userID,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
		return
	}

	bindToken := fmt.Sprintf("bt_%s_%d", appleUserID, time.Now().UnixNano())
	response.Success(c, gin.H{
		"bind_token":  bindToken,
		"apple_user_id": appleUserID,
		"need_bind":   true,
	})
}

// AppleBind Apple ID 绑定手机号
// POST /api/v1/auth/apple/bind
func (h *ThirdPartyHandler) AppleBind(c *gin.Context) {
	var req struct {
		BindToken string `json:"bind_token" binding:"required"`
		Phone     string `json:"phone" binding:"required"`
		CodeToken string `json:"code_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	userID := fmt.Sprintf("u_%s", req.Phone)
	accessToken, _ := h.jwtManager.GenerateAccessToken(userID, req.Phone)
	refreshToken, _ := h.jwtManager.GenerateRefreshToken(userID, req.Phone)

	response.Created(c, gin.H{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    7200,
	})
}

// Logout 登出
// POST /api/v1/auth/logout
func (h *ThirdPartyHandler) Logout(c *gin.Context) {
	// TODO: 将 Token 加入黑名单 (Redis)
	response.Success(c, gin.H{"message": "已登出"})
}
