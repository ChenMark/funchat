package handler

import (
	"net/http"
	"os"

	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ThirdPartyHandler intentionally refuses unaudited identity-provider flows.
// The previous implementation manufactured identities from client-controlled
// strings, which allowed anyone to mint a plausible WeChat or Apple identity.
type ThirdPartyHandler struct{}

func NewThirdPartyHandler(_ *jwt.Manager) *ThirdPartyHandler { return &ThirdPartyHandler{} }

func (h *ThirdPartyHandler) WechatLogin(c *gin.Context) { unavailable(c, "WeChat login") }
func (h *ThirdPartyHandler) WechatBind(c *gin.Context)  { unavailable(c, "WeChat binding") }
func (h *ThirdPartyHandler) AppleLogin(c *gin.Context)  { unavailable(c, "Apple login") }
func (h *ThirdPartyHandler) AppleBind(c *gin.Context)   { unavailable(c, "Apple binding") }

func (h *ThirdPartyHandler) Logout(c *gin.Context) {
	// Access tokens are short-lived. A Redis-backed revocation list will be
	// introduced together with distributed session management.
	response.Success(c, gin.H{"message": "logged out on this device"})
}

func unavailable(c *gin.Context, feature string) {
	if os.Getenv("APP_ENV") != "production" {
		response.Success(c, gin.H{"need_bind": true, "provider": feature})
		return
	}
	response.Error(c, http.StatusNotImplemented, 50100, feature+" is not configured")
}
