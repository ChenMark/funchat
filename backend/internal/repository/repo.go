package repository

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"funchat/backend/internal/model"

	"gorm.io/gorm"
)

// Repo 统一 repository
type Repo struct {
	DB *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{DB: db}
}

// ========== Auth ==========

// CreateVerificationCode 生成并存储验证码
func (r *Repo) CreateVerificationCode(phone string) (string, error) {
	// 生成 6 位验证码
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	code := fmt.Sprintf("%06d", n.Int64())

	// 查今日发送次数
	var todayCount int64
	today := time.Now().Format("2006-01-02")
	r.DB.Model(&model.VerificationCode{}).
		Where("phone = ? AND DATE(created_at) = ?", phone, today).
		Count(&todayCount)

	if todayCount >= 10 {
		return "", fmt.Errorf("daily_limit")
	}

	// 检查 60s 频控
	var recent model.VerificationCode
	if err := r.DB.Where("phone = ? AND created_at > ?", phone, time.Now().Add(-60*time.Second)).
		Order("created_at DESC").First(&recent).Error; err == nil {
		return "", fmt.Errorf("too_frequent")
	}

	vc := &model.VerificationCode{
		Phone:      phone,
		Code:       code,
		Type:       "code",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		DailyCount: int(todayCount) + 1,
	}
	if err := r.DB.Create(vc).Error; err != nil {
		return "", err
	}
	return code, nil
}

// VerifyCode 校验验证码，返回 code_token
func (r *Repo) VerifyCode(phone, code string) (string, bool, error) {
	var vc model.VerificationCode
	if err := r.DB.Where("phone = ? AND code = ? AND type = ? AND expires_at > ?",
		phone, code, "code", time.Now()).
		Order("created_at DESC").First(&vc).Error; err != nil {
		return "", false, nil // 验证码错误或过期
	}

	// 生成 code_token
	codeToken := fmt.Sprintf("ct_%s_%d", phone, time.Now().UnixNano())
	vc.CodeToken = codeToken
	vc.Type = "code_token"
	vc.ExpiresAt = time.Now().Add(10 * time.Minute)
	r.DB.Save(&vc)

	// 检查是否新用户
	var user model.User
	isNew := r.DB.Where("phone = ?", phone).First(&user).Error != nil

	return codeToken, isNew, nil
}

// VerifyCodeToken 校验 code_token
func (r *Repo) VerifyCodeToken(phone, codeToken string) bool {
	var vc model.VerificationCode
	if err := r.DB.Where("phone = ? AND code_token = ? AND type = ? AND expires_at > ?",
		phone, codeToken, "code_token", time.Now()).
		First(&vc).Error; err != nil {
		return false
	}
	return true
}

// CreateUser 创建用户
func (r *Repo) CreateUser(phone, nickname, avatar string) (*model.User, error) {
	if nickname == "" {
		nickname = fmt.Sprintf("用户%s", phone[7:])
	}
	user := &model.User{
		ID:       fmt.Sprintf("u_%s", phone),
		Nickname: nickname,
		Avatar:   avatar,
		Phone:    phone,
		Status:   1,
	}
	if err := r.DB.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByPhone 按手机号查用户
func (r *Repo) GetUserByPhone(phone string) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 按 ID 查用户
func (r *Repo) GetUserByID(id string) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// BindWechat 绑定微信
func (r *Repo) BindWechat(userID, unionID string) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).
		Update("wechat_union_id", unionID).Error
}

// ========== Friends ==========

// SearchUsers 搜索用户
func (r *Repo) SearchUsers(keyword, currentUserID string) ([]map[string]interface{}, error) {
	var users []model.User
	r.DB.Where("phone = ? OR nickname LIKE ?", keyword, "%"+keyword+"%").
		Limit(20).Find(&users)

	results := make([]map[string]interface{}, 0)
	for _, u := range users {
		if u.ID == currentUserID {
			continue
		}
		isFriend := r.IsFriend(currentUserID, u.ID)
		results = append(results, map[string]interface{}{
			"user_id":   u.ID,
			"nickname":  u.Nickname,
			"avatar":    u.Avatar,
			"is_friend": isFriend,
		})
	}
	return results, nil
}

// IsFriend 检查好友关系
func (r *Repo) IsFriend(userID, friendID string) bool {
	var count int64
	r.DB.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = 1", userID, friendID).
		Count(&count)
	return count > 0
}

// CreateFriendRequest 创建好友申请
func (r *Repo) CreateFriendRequest(fromID, toID, message string) (int64, error) {
	if fromID == toID {
		return 0, fmt.Errorf("self")
	}
	if r.IsFriend(fromID, toID) {
		return 0, fmt.Errorf("already_friend")
	}

	// 检查重复申请
	var count int64
	r.DB.Model(&model.FriendRequest{}).
		Where("from_user_id = ? AND to_user_id = ? AND status = 0", fromID, toID).
		Count(&count)
	if count > 0 {
		return 0, fmt.Errorf("duplicate")
	}

	req := &model.FriendRequest{
		FromUserID: fromID,
		ToUserID:   toID,
		Message:    message,
		Status:     0,
	}
	if err := r.DB.Create(req).Error; err != nil {
		return 0, err
	}
	return req.ID, nil
}

// GetPendingRequests 获取待处理申请
func (r *Repo) GetPendingRequests(userID string) ([]model.FriendRequest, error) {
	var reqs []model.FriendRequest
	r.DB.Where("to_user_id = ? AND status = 0", userID).Find(&reqs)
	return reqs, nil
}

// AcceptFriendRequest 同意申请
func (r *Repo) AcceptFriendRequest(requestID, userID string) error {
	var req model.FriendRequest
	if err := r.DB.Where("id = ? AND status = 0", requestID).First(&req).Error; err != nil {
		return fmt.Errorf("not_found")
	}
	if req.ToUserID != userID {
		return fmt.Errorf("forbidden")
	}

	// 更新申请状态
	r.DB.Model(&req).Update("status", 1)

	// 创建双向好友关系
	r.DB.Create(&model.Friendship{UserID: req.FromUserID, FriendID: req.ToUserID, Status: 1, Source: "search"})
	r.DB.Create(&model.Friendship{UserID: req.ToUserID, FriendID: req.FromUserID, Status: 1, Source: "search"})
	return nil
}

// RejectFriendRequest 拒绝申请
func (r *Repo) RejectFriendRequest(requestID, userID string) error {
	var req model.FriendRequest
	if err := r.DB.Where("id = ? AND status = 0", requestID).First(&req).Error; err != nil {
		return fmt.Errorf("not_found")
	}
	if req.ToUserID != userID {
		return fmt.Errorf("forbidden")
	}
	r.DB.Model(&req).Update("status", 2)
	return nil
}

// GetFriends 获取好友列表
func (r *Repo) GetFriends(userID string) ([]model.Friendship, error) {
	var friendships []model.Friendship
	r.DB.Where("user_id = ? AND status = 1", userID).Find(&friendships)
	return friendships, nil
}

// DeleteFriend 删除好友
func (r *Repo) DeleteFriend(userID, friendID string) error {
	r.DB.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("status", 2)
	r.DB.Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ?", friendID, userID).
		Update("status", 2)
	return nil
}

// ========== Messages ==========

// CreateMessage 创建消息
func (r *Repo) CreateMessage(fromID, toID string, msgType int8, content string, isBurn int8, burnDuration int) (*model.Message, error) {
	msg := &model.Message{
		FromUserID:   fromID,
		ToUserID:     toID,
		MsgType:      msgType,
		Content:      content,
		IsBurn:       isBurn,
		BurnDuration: burnDuration,
	}
	if err := r.DB.Create(msg).Error; err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *Repo) GetMessageByID(messageID int64) (*model.Message, error) {
	var message model.Message
	if err := r.DB.Where("id = ?", messageID).First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *Repo) ClearMessageContent(messageID int64) error {
	return r.DB.Model(&model.Message{}).Where("id = ?", messageID).
		Updates(map[string]interface{}{"content": "", "is_read": 1}).Error
}

// GetMessageHistory 消息历史
func (r *Repo) GetMessageHistory(userID, friendID string, page, size int) ([]model.Message, int64, error) {
	var messages []model.Message
	var total int64

	r.DB.Model(&model.Message{}).
		Where("((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
			userID, friendID, friendID, userID).
		Count(&total)

	offset := total - int64(page*size)
	if offset < 0 {
		offset = 0
	}
	r.DB.Where("((from_user_id = ? AND to_user_id = ?) OR (from_user_id = ? AND to_user_id = ?))",
		userID, friendID, friendID, userID).
		Order("created_at ASC").
		Offset(int(offset)).
		Limit(size).
		Find(&messages)

	return messages, total, nil
}

// MarkMessageRead 标记已读
func (r *Repo) MarkMessageRead(msgID, userID string) error {
	return r.DB.Model(&model.Message{}).
		Where("id = ? AND to_user_id = ?", msgID, userID).
		Update("is_read", 1).Error
}

// GetConversations 聊天列表
func (r *Repo) GetConversations(userID string) ([]map[string]interface{}, error) {
	type ConvResult struct {
		FriendID string
		LastMsg  string
		LastTs   time.Time
		Unread   int
	}

	var results []ConvResult
	r.DB.Raw(`
		SELECT 
			CASE WHEN from_user_id = ? THEN to_user_id ELSE from_user_id END AS friend_id,
			content AS last_msg,
			created_at AS last_ts,
			SUM(CASE WHEN to_user_id = ? AND is_read = 0 THEN 1 ELSE 0 END) AS unread
		FROM messages
		WHERE from_user_id = ? OR to_user_id = ?
		GROUP BY friend_id
		ORDER BY last_ts DESC
	`, userID, userID, userID, userID).Scan(&results)

	convs := make([]map[string]interface{}, 0)
	for _, r := range results {
		convs = append(convs, map[string]interface{}{
			"user_id":   r.FriendID,
			"nickname":  fmt.Sprintf("好友_%s", r.FriendID[2:6]),
			"last_msg":  r.LastMsg,
			"last_time": r.LastTs.Format("15:04"),
			"unread":    r.Unread,
		})
	}
	return convs, nil
}

// ========== Conditions ==========

// SetCondition 设置解锁条件
func (r *Repo) SetCondition(userID string, condType int8, isEnabled int8, params string) (int64, error) {
	cfg := &model.ConditionConfig{
		UserID:    userID,
		CondType:  condType,
		IsEnabled: isEnabled,
		Params:    params,
	}
	if err := r.DB.Create(cfg).Error; err != nil {
		return 0, err
	}
	return cfg.ID, nil
}

// GetConditions 获取用户条件
func (r *Repo) GetConditions(userID string) ([]model.ConditionConfig, error) {
	var conds []model.ConditionConfig
	r.DB.Where("user_id = ? AND is_enabled = 1", userID).Find(&conds)
	return conds, nil
}

// RecordUnlock 记录解锁
func (r *Repo) RecordUnlock(userID, targetID string, condType int8, success bool, detail string) {
	result := int8(2)
	if success {
		result = 1
	}
	r.DB.Create(&model.UnlockRecord{
		UserID:   userID,
		TargetID: targetID,
		CondType: condType,
		Result:   result,
		Detail:   detail,
	})
}

// ========== Quiz ==========

// CreateQuiz 创建题目
func (r *Repo) CreateQuiz(userID, question, answer string, maxTries int) (int64, error) {
	quiz := &model.QuizQuestion{
		UserID:   userID,
		Question: question,
		Answer:   answer,
		MaxTries: maxTries,
	}
	if err := r.DB.Create(quiz).Error; err != nil {
		return 0, err
	}
	return quiz.ID, nil
}

// GetQuiz 获取题目
func (r *Repo) GetQuiz(quizID int64) (*model.QuizQuestion, error) {
	var quiz model.QuizQuestion
	if err := r.DB.Where("id = ?", quizID).First(&quiz).Error; err != nil {
		return nil, err
	}
	return &quiz, nil
}

// GetQuizAttemptsToday 获取今日答题次数
func (r *Repo) GetQuizAttemptsToday(quizID int64) (int64, error) {
	var count int64
	today := time.Now().Format("2006-01-02")
	r.DB.Model(&model.UnlockRecord{}).
		Where("target_id = ? AND cond_type = 3 AND DATE(created_at) = ?", fmt.Sprintf("quiz_%d", quizID), today).
		Count(&count)
	return count, nil
}

// ========== Burn ==========

// CreateBurnMessage 创建焚毁消息记录
func (r *Repo) CreateBurnMessage(msgID int64, fromID, toID, content string, msgType int8, duration int) error {
	record := &model.BurnRecord{
		MessageID: msgID,
		UserID:    toID,
		Status:    0,
	}
	return r.DB.Create(record).Error
}

// GetBurnRecord 获取焚毁记录
func (r *Repo) GetBurnRecord(msgID int64) (*model.BurnRecord, error) {
	var record model.BurnRecord
	if err := r.DB.Where("message_id = ?", msgID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// StartBurnCountdown 开始倒计时
func (r *Repo) StartBurnCountdown(msgID int64, duration int) error {
	return r.DB.Model(&model.BurnRecord{}).
		Where("message_id = ?", msgID).
		Updates(map[string]interface{}{
			"status":  1,
			"read_at": time.Now(),
			"burn_at": time.Now().Add(time.Duration(duration) * time.Second),
		}).Error
}

// CompleteBurn 完成焚毁
func (r *Repo) CompleteBurn(msgID int64) error {
	return r.DB.Model(&model.BurnRecord{}).
		Where("message_id = ?", msgID).
		Updates(map[string]interface{}{
			"status": 2,
		}).Error
}

func (r *Repo) DestroyExpiredBurns(now time.Time) error {
	tx := r.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Exec(`
		UPDATE messages m
		JOIN burn_records b ON b.message_id = m.id
		SET m.content = '', m.is_read = 1
		WHERE b.status = 1 AND b.burn_at <= ?`, now).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&model.BurnRecord{}).
		Where("status = ? AND burn_at <= ?", 1, now).
		Update("status", 2).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// GetPendingBurns 获取待销毁消息
func (r *Repo) GetPendingBurns(userID string) ([]model.BurnRecord, error) {
	var records []model.BurnRecord
	r.DB.Where("user_id = ? AND status = 0", userID).Find(&records)
	return records, nil
}
