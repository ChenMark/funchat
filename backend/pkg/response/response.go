package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse 统一 API 响应格式
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应 (200)
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Created 创建成功 (201)
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, APIResponse{
		Code:    code,
		Message: message,
	})
}

// BadRequest 参数错误 (400)
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 40000, message)
}

// Unauthorized 未授权 (401)
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, 40100, message)
}

// Forbidden 禁止 (403)
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, 40300, message)
}

// NotFound 未找到 (404)
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 40400, message)
}

// Conflict 冲突 (409)
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, 40900, message)
}

// InternalError 内部错误 (500)
func InternalError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, 50000, "服务器内部错误")
}
