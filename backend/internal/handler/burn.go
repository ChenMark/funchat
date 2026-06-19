package handler

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"funchat/backend/internal/repository"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// BurnHandler 阅后即焚 handler (BURN-001/002)
type BurnHandler struct {
	mu          sync.RWMutex
	repo        *repository.Repo
	burnRecords map[int64]*BurnRecordItem // 内存 (即时倒计时)
	burnIDSeq   int64
}

// BurnRecordItem 焚毁记录
type BurnRecordItem struct {
	ID           int64     `json:"id"`
	MessageID    int64     `json:"message_id"`
	FromUserID   string    `json:"from_user_id"`
	ToUserID     string    `json:"to_user_id"`
	Content      string    `json:"content"`
	MsgType      int8      `json:"msg_type"`
	Duration     int       `json:"duration"`      // 倒计时秒数
	Status       int8      `json:"status"`        // 0=未读 1=已读(倒计时) 2=已焚毁
	ReadAt       time.Time `json:"read_at,omitempty"`
	BurnAt       time.Time `json:"burn_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewBurnHandler 创建 BurnHandler
func NewBurnHandler(repo *repository.Repo) *BurnHandler {
	return &BurnHandler{
		repo:        repo,
		burnRecords: make(map[int64]*BurnRecordItem),
	}
}

// SendBurnMessage 发送阅后即焚消息
// POST /api/v1/messages/send-burn
func (h *BurnHandler) SendBurnMessage(c *gin.Context) {
	var req struct {
		ToUserID string `json:"to_user_id" binding:"required"`
		MsgType  int8   `json:"msg_type"`
		Content  string `json:"content" binding:"required"`
		Duration int    `json:"duration"` // 倒计时秒数 (默认 5)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Duration <= 0 {
		req.Duration = 5
	}
	if req.Duration > 60 {
		req.Duration = 60
	}
	if req.MsgType == 0 {
		req.MsgType = 1
	}

	currentUserID := c.GetString("user_id")

	h.mu.Lock()
	h.burnIDSeq++
	record := &BurnRecordItem{
		ID:         h.burnIDSeq,
		MessageID:  time.Now().UnixNano(),
		FromUserID: currentUserID,
		ToUserID:   req.ToUserID,
		Content:    req.Content,
		MsgType:    req.MsgType,
		Duration:   req.Duration,
		Status:     0,
		CreatedAt:  time.Now(),
	}
	h.burnRecords[record.MessageID] = record
	h.mu.Unlock()

	response.Created(c, gin.H{
		"msg_id":   record.MessageID,
		"duration": req.Duration,
		"status":   "sent",
	})
}

// ReadBurnMessage 读取阅后即焚消息（开始倒计时）
// POST /api/v1/messages/:id/burn-read
func (h *BurnHandler) ReadBurnMessage(c *gin.Context) {
	msgID := parseID(c.Param("id"))
	currentUserID := c.GetString("user_id")

	h.mu.Lock()
	record, exists := h.burnRecords[msgID]
	if !exists {
		h.mu.Unlock()
		response.NotFound(c, "消息不存在或已焚毁")
		return
	}
	if record.ToUserID != currentUserID && record.FromUserID != currentUserID {
		h.mu.Unlock()
		response.Forbidden(c, "无权查看此消息")
		return
	}
	// 只有接收方才能读取（触发倒计时）
	if record.ToUserID != currentUserID {
		h.mu.Unlock()
		// 发送方查看状态
		response.Success(c, gin.H{
			"msg_id":   msgID,
			"status":   "sent",
			"duration": record.Duration,
		})
		return
	}
	if record.Status != 0 {
		h.mu.Unlock()
		response.Error(c, http.StatusGone, 41000, "消息已焚毁")
		return
	}

	// 开始倒计时
	record.Status = 1
	record.ReadAt = time.Now()
	record.BurnAt = time.Now().Add(time.Duration(record.Duration) * time.Second)
	duration := record.Duration // 捕获到局部变量
	h.mu.Unlock()

	// 定时销毁 (异步) — 使用局部变量避免闭包引用问题
	go func() {
		time.Sleep(time.Duration(duration) * time.Second)
		h.mu.Lock()
		if r, ok := h.burnRecords[msgID]; ok && r.Status == 1 {
			r.Status = 2
			r.Content = "" // 硬删除内容
		}
		h.mu.Unlock()
	}()

	response.Success(c, gin.H{
		"msg_id":      msgID,
		"content":     record.Content,
		"msg_type":    record.MsgType,
		"duration":    record.Duration,
		"burn_at":     record.BurnAt.Unix(),
		"status":      "reading",
	})
}

// BurnStatus 查询焚毁状态
// GET /api/v1/messages/:id/burn-status
func (h *BurnHandler) BurnStatus(c *gin.Context) {
	msgID := parseID(c.Param("id"))

	h.mu.RLock()
	record, exists := h.burnRecords[msgID]
	h.mu.RUnlock()

	if !exists {
		response.NotFound(c, "消息不存在")
		return
	}

	var remaining int64
	if record.Status == 1 {
		remaining = int64(record.BurnAt.Sub(time.Now()).Seconds())
		if remaining < 0 {
			remaining = 0
		}
	}

	response.Success(c, gin.H{
		"msg_id":    msgID,
		"status":    record.Status,
		"remaining": remaining,
		"burn_at":   record.BurnAt.Unix(),
	})
}

// DestroyBurnMessage 强制销毁（服务端主动触发）
// POST /api/v1/messages/:id/burn-destroy
func (h *BurnHandler) DestroyBurnMessage(c *gin.Context) {
	msgID := parseID(c.Param("id"))

	h.mu.Lock()
	record, exists := h.burnRecords[msgID]
	if !exists {
		h.mu.Unlock()
		response.NotFound(c, "消息不存在")
		return
	}

	record.Status = 2
	record.Content = "" // 硬删除
	h.mu.Unlock()

	response.Success(c, gin.H{
		"msg_id":  msgID,
		"message": "消息已焚毁",
	})
}

// CleanupExpired 清理已过期焚毁消息 (定时任务)
func (h *BurnHandler) CleanupExpired() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, record := range h.burnRecords {
		if record.Status == 2 && time.Since(record.BurnAt) > 24*time.Hour {
			delete(h.burnRecords, id)
		}
	}
}

// GetPendingBurns 获取待销毁消息列表 (崩溃恢复)
// GET /api/v1/burns/pending
func (h *BurnHandler) GetPendingBurns(c *gin.Context) {
	currentUserID := c.GetString("user_id")

	h.mu.RLock()
	var pending []BurnRecordItem
	for _, record := range h.burnRecords {
		if record.ToUserID == currentUserID && record.Status == 0 {
			pending = append(pending, *record)
		}
	}
	h.mu.RUnlock()

	if pending == nil {
		pending = []BurnRecordItem{}
	}

	response.Success(c, gin.H{
		"pending": pending,
	})
}

// --- 定时清理 goroutine ---

// StartBurnCleanup 启动焚毁清理定时任务
func (h *BurnHandler) StartBurnCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			h.CleanupExpired()
		}
	}()
	fmt.Println("[BURN] 清理定时任务已启动")
}
