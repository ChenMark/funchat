package handler

import (
	"net/http"
	"strconv"
	"strings"

	"funchat/backend/internal/model"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

const maxMessageBytes = 4 * 1024

type MessageHandler struct {
	Hub  *Hub
	repo *repository.Repo
}

func NewMessageHandler(hub *Hub, repo *repository.Repo) *MessageHandler {
	return &MessageHandler{Hub: hub, repo: repo}
}

// SendMessage persists before notifying the recipient. The message ID returned
// here is the only delivery ID that clients should treat as authoritative.
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req struct {
		ToUserID     string `json:"to_user_id" binding:"required"`
		MsgType      int8   `json:"msg_type"`
		Content      string `json:"content" binding:"required"`
		IsBurn       int8   `json:"is_burn"`
		BurnDuration int    `json:"burn_duration"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid message request")
		return
	}
	if len(strings.TrimSpace(req.Content)) == 0 || len(req.Content) > maxMessageBytes {
		response.BadRequest(c, "message content must be 1-4096 bytes")
		return
	}
	if req.MsgType == 0 { req.MsgType = 1 }
	if req.MsgType < 1 || req.MsgType > 3 {
		response.BadRequest(c, "unsupported message type")
		return
	}

	fromUserID := c.GetString("user_id")
	if fromUserID == req.ToUserID || !h.repo.IsFriend(fromUserID, req.ToUserID) {
		response.Forbidden(c, "messages can only be sent to a friend")
		return
	}
	msg, err := h.repo.CreateMessage(fromUserID, req.ToUserID, req.MsgType, req.Content, req.IsBurn, req.BurnDuration)
	if err != nil {
		response.InternalError(c)
		return
	}
	h.Hub.SendToUser(req.ToUserID, &WSMessage{
		Type: "chat", From: fromUserID, To: req.ToUserID, MsgType: req.MsgType,
		Content: req.Content, MsgID: msg.ID, Timestamp: msg.CreatedAt.Unix(),
	})
	response.Created(c, gin.H{"msg_id": msg.ID, "timestamp": msg.CreatedAt.Unix()})
}

func (h *MessageHandler) MarkRead(c *gin.Context) {
	if err := h.repo.MarkMessageRead(c.Param("id"), c.GetString("user_id")); err != nil {
		response.InternalError(c)
		return
	}
	response.Success(c, gin.H{"message": "marked read"})
}

func (h *MessageHandler) GetHistory(c *gin.Context) {
	friendID := c.Query("friend_id")
	if friendID == "" {
		response.BadRequest(c, "friend_id is required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if size < 1 { size = 20 }
	if size > 50 { size = 50 }
	messages, total, err := h.repo.GetMessageHistory(c.GetString("user_id"), friendID, page, size)
	if err != nil { response.InternalError(c); return }
	if messages == nil { messages = []model.Message{} }
	response.Success(c, gin.H{"messages": messages, "page": page, "size": size, "total": total, "has_more": int64(page*size) < total})
}

func (h *MessageHandler) GetConversations(c *gin.Context) {
	conversations, err := h.repo.GetConversations(c.GetString("user_id"))
	if err != nil { response.InternalError(c); return }
	response.Success(c, gin.H{"conversations": conversations})
}

// Media uploads must be implemented through signed object-storage uploads.
// Returning a made-up URL is worse than a clear failure because it loses user data.
func (h *MessageHandler) UploadImage(c *gin.Context) { mediaUnavailable(c) }
func (h *MessageHandler) UploadVoice(c *gin.Context) { mediaUnavailable(c) }

func mediaUnavailable(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, 50101, "media upload is not configured")
}
