package handler

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sync"
	"time"

	"funchat/backend/internal/model"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ConditionHandler 解锁条件 handler (COND + LOC + STEP + QUIZ)
type ConditionHandler struct {
	mu             sync.RWMutex
	repo           *repository.Repo
	conditions     map[string][]*ConditionConfig // 内存缓存 (即时条件消息)
	quizAttempts   map[int64]int                 // quizID -> attempts (内存,每日重置)
	quizResetDay   map[int64]string              // quizID -> 重置日期
	conditionMsgs  map[int64]*ConditionMsg       // msgID -> condition info
}

// ConditionConfig 解锁条件配置
type ConditionConfig struct {
	ID        int64  `json:"id"`
	UserID    string `json:"user_id"`
	CondType  int8   `json:"cond_type"`  // 1=定位 2=步数 3=答题
	IsEnabled int8   `json:"is_enabled"`
	Params    string `json:"params"`     // JSON
}

// ConditionMsg 条件消息
type ConditionMsg struct {
	MsgID       int64    `json:"msg_id"`
	FromUserID  string   `json:"from_user_id"`
	ToUserID    string   `json:"to_user_id"`
	Content     string   `json:"content"`
	CondTypes   []int8   `json:"cond_types"` // 需要满足的条件
	Status      string   `json:"status"`     // pending/verified/unlocked
	CreatedAt   int64    `json:"created_at"`
}

// NewConditionHandler 创建
func NewConditionHandler(repo *repository.Repo) *ConditionHandler {
	return &ConditionHandler{
		repo:           repo,
		conditions:     make(map[string][]*ConditionConfig),
		quizAttempts:   make(map[int64]int),
		quizResetDay:   make(map[int64]string),
		conditionMsgs:  make(map[int64]*ConditionMsg),
	}
}

// ========== COND-001: 条件消息发送 ==========

// SendConditionalMessage 发送条件消息
// POST /api/v1/messages/send-conditional
func (h *ConditionHandler) SendConditionalMessage(c *gin.Context) {
	var req struct {
		ToUserID  string `json:"to_user_id" binding:"required"`
		Content   string `json:"content" binding:"required"`
		CondTypes []int8 `json:"cond_types" binding:"required"` // [1=定位, 2=步数, 3=答题]
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if len(req.CondTypes) == 0 {
		response.BadRequest(c, "至少选择一种解锁条件")
		return
	}

	currentUserID := c.GetString("user_id")

	h.mu.Lock()
	msgID := time.Now().UnixNano()
	condMsg := &ConditionMsg{
		MsgID:      msgID,
		FromUserID: currentUserID,
		ToUserID:   req.ToUserID,
		Content:    req.Content,
		CondTypes:  req.CondTypes,
		Status:     "pending",
		CreatedAt:  time.Now().Unix(),
	}
	h.conditionMsgs[msgID] = condMsg
	h.mu.Unlock()

	response.Created(c, gin.H{
		"msg_id":     msgID,
		"cond_types": req.CondTypes,
		"status":     "pending",
	})
}

// ========== COND-002: 条件消息状态机 ==========

// GetConditionStatus 查询条件消息状态
// GET /api/v1/messages/:id/condition-status
func (h *ConditionHandler) GetConditionStatus(c *gin.Context) {
	msgID := parseID(c.Param("id"))

	h.mu.RLock()
	condMsg, exists := h.conditionMsgs[msgID]
	h.mu.RUnlock()

	if !exists {
		response.NotFound(c, "消息不存在")
		return
	}

	// 检查每个条件是否满足
	type CondStatus struct {
		CondType int8   `json:"cond_type"`
		Label    string `json:"label"`
		Met      bool   `json:"met"`
	}

	statuses := make([]CondStatus, 0)
	for _, ct := range condMsg.CondTypes {
		met := h.checkCondition(condMsg.ToUserID, condMsg.FromUserID, ct)
		label := condTypeLabel(ct)
		statuses = append(statuses, CondStatus{CondType: ct, Label: label, Met: met})
	}

	// 任一条件满足 → unlocked
	allMet := false
	for _, s := range statuses {
		if s.Met {
			allMet = true
			break
		}
	}

	if allMet && condMsg.Status == "pending" {
		h.mu.Lock()
		condMsg.Status = "unlocked"
		h.mu.Unlock()
	}

	response.Success(c, gin.H{
		"msg_id":      msgID,
		"status":      condMsg.Status,
		"conditions":  statuses,
		"any_met":     allMet,
	})
}

// ========== COND-003: 消息撤回 ==========

// RevokeMessage 撤回消息
// POST /api/v1/messages/:id/revoke
func (h *ConditionHandler) RevokeMessage(c *gin.Context) {
	msgID := parseID(c.Param("id"))
	currentUserID := c.GetString("user_id")

	h.mu.RLock()
	condMsg, exists := h.conditionMsgs[msgID]
	h.mu.RUnlock()

	if !exists {
		response.NotFound(c, "消息不存在")
		return
	}
	if condMsg.FromUserID != currentUserID {
		response.Forbidden(c, "只能撤回自己的消息")
		return
	}

	// 2分钟撤回限制
	if time.Now().Unix()-condMsg.CreatedAt > 120 {
		response.Error(c, http.StatusBadRequest, 40040, "超过2分钟无法撤回")
		return
	}

	h.mu.Lock()
	condMsg.Status = "revoked"
	h.mu.Unlock()

	response.Success(c, gin.H{"message": "消息已撤回", "msg_id": msgID})
}

// ========== LOC-004: 定位校验 API ==========

// VerifyLocation 定位校验
// POST /api/v1/conditions/verify-location
func (h *ConditionHandler) VerifyLocation(c *gin.Context) {
	var req struct {
		TargetUserID string  `json:"target_user_id" binding:"required"`
		Lat          float64 `json:"lat" binding:"required"`
		Lng          float64 `json:"lng" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	currentUserID := c.GetString("user_id")

	// TODO: 查询对方设置的条件参数
	// 模拟：对方要求 500m 内
	targetLat := req.Lat + 0.001 // 模拟对方位置
	targetLng := req.Lng + 0.001

	distance := haversine(req.Lat, req.Lng, targetLat, targetLng)
	withinRange := distance <= 500

	// 记录解锁记录
	h.recordUnlock(currentUserID, req.TargetUserID, 1, withinRange, fmt.Sprintf("距离: %.0fm", distance))

	response.Success(c, gin.H{
		"distance":     int(distance),
		"within_range": withinRange,
		"required":     500,
		"unit":         "m",
	})
}

// ========== LOC-005: 虚拟定位检测 ==========

// DetectFakeLocation 虚拟定位检测
func (h *ConditionHandler) DetectFakeLocation(c *gin.Context) {
	var req struct {
		Lat     float64 `json:"lat" binding:"required"`
		Lng     float64 `json:"lng" binding:"required"`
		Accuracy float64 `json:"accuracy"`
		IsMock   bool    `json:"is_mock"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 检测规则
	suspicious := false
	reasons := []string{}

	if req.IsMock {
		suspicious = true
		reasons = append(reasons, "检测到模拟定位")
	}
	if req.Accuracy == 0 || req.Accuracy > 100 {
		suspicious = true
		reasons = append(reasons, "定位精度异常")
	}
	if req.Lat == 0 && req.Lng == 0 {
		suspicious = true
		reasons = append(reasons, "坐标为空")
	}

	response.Success(c, gin.H{
		"suspicious": suspicious,
		"reasons":    reasons,
		"pass":       !suspicious,
	})
}

// ========== STEP-004: 步数校验 API ==========

// VerifySteps 步数校验
// POST /api/v1/conditions/verify-steps
func (h *ConditionHandler) VerifySteps(c *gin.Context) {
	var req struct {
		TargetUserID string `json:"target_user_id" binding:"required"`
		Steps        int    `json:"steps" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	currentUserID := c.GetString("user_id")

	// TODO: 查询对方设置的目标步数
	requiredSteps := 8000 // 默认

	met := req.Steps >= requiredSteps
	h.recordUnlock(currentUserID, req.TargetUserID, 2, met, fmt.Sprintf("步数: %d/%d", req.Steps, requiredSteps))

	response.Success(c, gin.H{
		"current":  req.Steps,
		"required": requiredSteps,
		"met":      met,
	})
}

// ========== STEP-005: 步数防作弊 ==========

// DetectStepCheating 步数防作弊检测
func (h *ConditionHandler) DetectStepCheating(c *gin.Context) {
	var req struct {
		Steps      int   `json:"steps"`
		StartTime  int64 `json:"start_time"`
		EndTime    int64 `json:"end_time"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	suspicious := false
	reasons := []string{}

	// 检测规则
	durationHours := float64(req.EndTime-req.StartTime) / 3600
	if durationHours > 0 {
		stepsPerHour := float64(req.Steps) / durationHours
		if stepsPerHour > 20000 {
			suspicious = true
			reasons = append(reasons, "步数增速异常")
		}
	}
	if req.Steps > 100000 {
		suspicious = true
		reasons = append(reasons, "单日步数异常")
	}

	response.Success(c, gin.H{
		"suspicious": suspicious,
		"reasons":    reasons,
		"pass":       !suspicious,
	})
}

// ========== QUIZ-002: 答题校验 API ==========

// QuizQuestion 题目信息
type QuizQuestion struct {
	ID       int64  `json:"id"`
	UserID   string `json:"user_id"`
	Question string `json:"question"`
	Answer   string `json:"answer"` // 仅出题人可见
	MaxTries int    `json:"max_tries"`
}

// SetQuiz 设置题目
// POST /api/v1/conditions/set-quiz
func (h *ConditionHandler) SetQuiz(c *gin.Context) {
	var req struct {
		Question string `json:"question" binding:"required"`
		Answer   string `json:"answer" binding:"required"`
		MaxTries int    `json:"max_tries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.MaxTries <= 0 {
		req.MaxTries = 3
	}

	currentUserID := c.GetString("user_id")

	quizID, err := h.repo.CreateQuiz(currentUserID, req.Question, req.Answer, req.MaxTries)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Created(c, gin.H{
		"quiz_id":   quizID,
		"question":  req.Question,
		"max_tries": req.MaxTries,
	})
}

// VerifyQuiz 答题校验
// POST /api/v1/conditions/verify-quiz
func (h *ConditionHandler) VerifyQuiz(c *gin.Context) {
	var req struct {
		QuizID int64  `json:"quiz_id" binding:"required"`
		Answer string `json:"answer" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	q, err := h.repo.GetQuiz(req.QuizID)
	if err != nil {
		response.NotFound(c, "题目不存在")
		return
	}

	currentUserID := c.GetString("user_id")

	// 次数检查 + 每日自动重置
	h.mu.Lock()
	today := time.Now().Format("2006-01-02")
	if h.quizResetDay[req.QuizID] != today {
		// 新的一天，重置次数
		h.quizAttempts[req.QuizID] = 0
		h.quizResetDay[req.QuizID] = today
	}
	attempts := h.quizAttempts[req.QuizID]
	if attempts >= q.MaxTries {
		h.mu.Unlock()
		response.Error(c, http.StatusTooManyRequests, 40050, "答题次数已用完，明天再来试试吧")
		return
	}
	h.quizAttempts[req.QuizID] = attempts + 1
	h.mu.Unlock()

	correct := q.Answer == req.Answer
	h.recordUnlock(currentUserID, q.UserID, 3, correct, fmt.Sprintf("答题: %v", correct))

	response.Success(c, gin.H{
		"correct":       correct,
		"attempts_used": attempts + 1,
		"max_tries":     q.MaxTries,
	})
}

// GetQuizQuestion 获取题目（隐藏答案）
// GET /api/v1/conditions/quiz/:id
func (h *ConditionHandler) GetQuizQuestion(c *gin.Context) {
	id := parseID(c.Param("id"))
	q, err := h.repo.GetQuiz(id)
	if err != nil {
		response.NotFound(c, "题目不存在")
		return
	}

	response.Success(c, gin.H{
		"quiz_id":   q.ID,
		"question":  q.Question,
		"max_tries": q.MaxTries,
	})
}

// ========== 条件配置管理 ==========

// SetConditions 设置解锁条件
// POST /api/v1/conditions/set
func (h *ConditionHandler) SetConditions(c *gin.Context) {
	var req struct {
		CondType  int8   `json:"cond_type" binding:"required"`
		IsEnabled int8   `json:"is_enabled"`
		Params    string `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	currentUserID := c.GetString("user_id")

	condID, err := h.repo.SetCondition(currentUserID, req.CondType, req.IsEnabled, req.Params)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Created(c, gin.H{
		"condition_id": condID,
		"cond_type":    condTypeLabel(req.CondType),
	})
}

// GetConditions 获取用户的解锁条件配置
// GET /api/v1/conditions/:user_id
func (h *ConditionHandler) GetConditions(c *gin.Context) {
	userID := c.Param("user_id")

	conds, err := h.repo.GetConditions(userID)
	if err != nil {
		conds = []model.ConditionConfig{}
	}

	type CondWithStatus struct {
		ID        int64  `json:"id"`
		CondType  int8   `json:"cond_type"`
		Label     string `json:"label"`
		IsEnabled int8   `json:"is_enabled"`
		Params    string `json:"params"`
	}

	result := make([]CondWithStatus, 0)
	for _, c := range conds {
		result = append(result, CondWithStatus{
			ID: c.ID, CondType: c.CondType, Label: condTypeLabel(c.CondType),
			IsEnabled: c.IsEnabled, Params: c.Params,
		})
	}

	response.Success(c, gin.H{"conditions": result})
}

// ========== 辅助方法 ==========

func (h *ConditionHandler) checkCondition(userID, targetID string, condType int8) bool {
	// TODO: 实际从数据库/传感器读取真实数据
	switch condType {
	case 1: // 定位
		return true // 模拟
	case 2: // 步数
		return true // 模拟
	case 3: // 答题
		return false // 模拟未答题
	}
	return false
}

func (h *ConditionHandler) recordUnlock(userID, targetID string, condType int8, success bool, detail string) {
	if h.repo != nil {
		h.repo.RecordUnlock(userID, targetID, condType, success, detail)
	}
	fmt.Printf("[UNLOCK] user=%s target=%s type=%d success=%v detail=%s\n",
		userID, targetID, condType, success, detail)
}

func condTypeLabel(t int8) string {
	switch t {
	case 1:
		return "定位解锁"
	case 2:
		return "步数解锁"
	case 3:
		return "答题解锁"
	default:
		return "未知"
	}
}

func parseID(s string) int64 {
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}

// haversine 计算两点距离（米）
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// suppress unused imports
var _ = rand.Reader
var _ = big.NewInt
var _ = http.StatusOK
