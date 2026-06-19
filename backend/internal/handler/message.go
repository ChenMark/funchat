package handler

import (
	"fmt"
	"strconv"
	"time"

	"funchat/backend/internal/model"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// MessageHandler 消息 handler
type MessageHandler struct {
	Hub  *Hub
	repo *repository.Repo
}

// NewMessageHandler 创建 MessageHandler
func NewMessageHandler(hub *Hub, repo *repository.Repo) *MessageHandler {
	return &MessageHandler{Hub: hub, repo: repo}
}

// SendMessage 发送消息
// POST /api/v1/messages/send
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req struct {
		ToUserID     string `json:"to_user_id" binding:"required"`
		MsgType      int8   `json:"msg_type"`
		Content      string `json:"content" binding:"required"`
		IsBurn       int8   `json:"is_burn"`
		BurnDuration int    `json:"burn_duration"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.MsgType == 0 {
		req.MsgType = 1
	}

	currentUserID := c.GetString("user_id")

	msg, err := h.repo.CreateMessage(currentUserID, req.ToUserID, req.MsgType, req.Content, req.IsBurn, req.BurnDuration)
	if err != nil {
		response.InternalError(c)
		return
	}

	// WebSocket 推送
	wsMsg := &WSMessage{
		Type:      "chat",
		From:      currentUserID,
		To:        req.ToUserID,
		MsgType:   req.MsgType,
		Content:   req.Content,
		MsgID:     msg.ID,
		Timestamp: msg.CreatedAt.Unix(),
	}
	h.Hub.SendToUser(req.ToUserID, wsMsg)

	response.Created(c, gin.H{
		"msg_id":    msg.ID,
		"timestamp": msg.CreatedAt.Unix(),
	})
}

// MarkRead 标记已读
// POST /api/v1/messages/:id/read
func (h *MessageHandler) MarkRead(c *gin.Context) {
	msgIDStr := c.Param("id")
	currentUserID := c.GetString("user_id")

	h.repo.MarkMessageRead(msgIDStr, currentUserID)

	response.Success(c, gin.H{"message": "已标记已读"})
}

// GetHistory 消息历史
// GET /api/v1/messages/history?friend_id=xxx&page=1&size=20
func (h *MessageHandler) GetHistory(c *gin.Context) {
	friendID := c.Query("friend_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size > 50 {
		size = 50
	}

	currentUserID := c.GetString("user_id")

	messages, total, err := h.repo.GetMessageHistory(currentUserID, friendID, page, size)
	if err != nil {
		messages = []model.Message{}
	}

	response.Success(c, gin.H{
		"messages": messages,
		"page":     page,
		"size":     size,
		"total":    total,
		"has_more": int64(page*size) < total,
	})
}

// GetConversations 聊天列表
// GET /api/v1/conversations
func (h *MessageHandler) GetConversations(c *gin.Context) {
	currentUserID := c.GetString("user_id")

	convs, err := h.repo.GetConversations(currentUserID)
	if err != nil {
		convs = []map[string]interface{}{}
	}

	response.Success(c, gin.H{
		"conversations": convs,
	})
}

// UploadImage 图片上传
// POST /api/v1/messages/upload-image
func (h *MessageHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		response.BadRequest(c, "请选择图片")
		return
	}

	// TODO: 上传到腾讯云 COS
	imageURL := fmt.Sprintf("https://cos.funchat.com/images/%d_%s", time.Now().Unix(), file.Filename)
	thumbnailURL := imageURL + "?thumbnail=200x200"

	response.Created(c, gin.H{
		"image_url":     imageURL,
		"thumbnail_url": thumbnailURL,
		"file_size":     file.Size,
	})
}

// UploadVoice 语音上传
// POST /api/v1/messages/upload-voice
func (h *MessageHandler) UploadVoice(c *gin.Context) {
	file, err := c.FormFile("voice")
	if err != nil {
		response.BadRequest(c, "请上传语音文件")
		return
	}

	voiceURL := fmt.Sprintf("https://cos.funchat.com/voice/%d_%s", time.Now().Unix(), file.Filename)

	response.Created(c, gin.H{
		"voice_url": voiceURL,
		"duration":  0,
		"file_size": file.Size,
	})
}
