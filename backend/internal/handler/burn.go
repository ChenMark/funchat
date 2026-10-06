package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"funchat/backend/internal/model"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// BurnHandler stores the message and destruction state in MySQL. It has no
// in-process records, so a restart cannot resurrect a supposedly destroyed
// message or lose a pending timer.
type BurnHandler struct{ repo *repository.Repo }

func NewBurnHandler(repo *repository.Repo) *BurnHandler { return &BurnHandler{repo: repo} }

func (h *BurnHandler) SendBurnMessage(c *gin.Context) {
	var req struct {
		ToUserID string `json:"to_user_id" binding:"required"`
		MsgType  int8   `json:"msg_type"`
		Content  string `json:"content" binding:"required"`
		Duration int    `json:"duration"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(strings.TrimSpace(req.Content)) == 0 || len(req.Content) > maxMessageBytes {
		response.BadRequest(c, "invalid burn message")
		return
	}
	if req.Duration <= 0 { req.Duration = 5 }
	if req.Duration > 60 { req.Duration = 60 }
	if req.MsgType == 0 { req.MsgType = 1 }
	fromUserID := c.GetString("user_id")
	if fromUserID == req.ToUserID || !h.repo.IsFriend(fromUserID, req.ToUserID) {
		response.Forbidden(c, "messages can only be sent to a friend")
		return
	}
	message, err := h.repo.CreateMessage(fromUserID, req.ToUserID, req.MsgType, req.Content, 1, req.Duration)
	if err != nil || h.repo.CreateBurnMessage(message.ID, fromUserID, req.ToUserID, req.Content, req.MsgType, req.Duration) != nil {
		response.InternalError(c)
		return
	}
	response.Created(c, gin.H{"msg_id": message.ID, "duration": req.Duration, "status": "sent"})
}

func (h *BurnHandler) ReadBurnMessage(c *gin.Context) {
	message, record, ok := h.load(c)
	if !ok { return }
	userID := c.GetString("user_id")
	if message.ToUserID != userID {
		response.Forbidden(c, "only the recipient can open this message")
		return
	}
	if record.Status == 2 || (record.Status == 1 && time.Now().After(record.BurnAt)) {
		_ = h.repo.CompleteBurn(message.ID)
		_ = h.repo.ClearMessageContent(message.ID)
		response.Error(c, http.StatusGone, 41000, "message has been destroyed")
		return
	}
	if record.Status == 0 {
		if err := h.repo.StartBurnCountdown(message.ID, message.BurnDuration); err != nil {
			response.InternalError(c)
			return
		}
		record, _ = h.repo.GetBurnRecord(message.ID)
	}
	response.Success(c, gin.H{"msg_id": message.ID, "content": message.Content, "msg_type": message.MsgType, "burn_at": record.BurnAt.Unix(), "status": "reading"})
}

func (h *BurnHandler) BurnStatus(c *gin.Context) {
	message, record, ok := h.load(c)
	if !ok { return }
	userID := c.GetString("user_id")
	if userID != message.FromUserID && userID != message.ToUserID { response.Forbidden(c, "not allowed"); return }
	if record.Status == 1 && time.Now().After(record.BurnAt) {
		_ = h.repo.CompleteBurn(message.ID)
		_ = h.repo.ClearMessageContent(message.ID)
		record, _ = h.repo.GetBurnRecord(message.ID)
	}
	remaining := int64(0)
	if record.Status == 1 { remaining = max(int64(time.Until(record.BurnAt).Seconds()), 0) }
	response.Success(c, gin.H{"msg_id": message.ID, "status": record.Status, "remaining": remaining, "burn_at": record.BurnAt.Unix()})
}

func (h *BurnHandler) DestroyBurnMessage(c *gin.Context) {
	message, _, ok := h.load(c)
	if !ok { return }
	if c.GetString("user_id") != message.FromUserID { response.Forbidden(c, "only the sender can destroy this message"); return }
	if err := h.repo.CompleteBurn(message.ID); err != nil { response.InternalError(c); return }
	if err := h.repo.ClearMessageContent(message.ID); err != nil { response.InternalError(c); return }
	response.Success(c, gin.H{"msg_id": message.ID, "message": "destroyed"})
}

func (h *BurnHandler) GetPendingBurns(c *gin.Context) {
	records, err := h.repo.GetPendingBurns(c.GetString("user_id"))
	if err != nil { response.InternalError(c); return }
	response.Success(c, gin.H{"pending": records})
}

func (h *BurnHandler) StartBurnCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := h.repo.DestroyExpiredBurns(time.Now()); err != nil {
				// A later tick retries safely; state remains in the database.
				continue
			}
		}
	}()
}

func (h *BurnHandler) load(c *gin.Context) (*model.Message, *model.BurnRecord, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "invalid message id")
		return nil, nil, false
	}
	message, err := h.repo.GetMessageByID(id)
	if err != nil || message.IsBurn != 1 {
		response.NotFound(c, "burn message not found")
		return nil, nil, false
	}
	record, err := h.repo.GetBurnRecord(id)
	if err != nil {
		response.NotFound(c, "burn record not found")
		return nil, nil, false
	}
	return message, record, true
}
