package model

import "time"

// User 用户表
type User struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	Nickname     string    `json:"nickname" gorm:"size:20;default:''"`
	Avatar       string    `json:"avatar" gorm:"size:500;default:''"`
	Phone        string    `json:"phone" gorm:"uniqueIndex;size:11;default:''"`
	WechatUnionID string   `json:"wechat_union_id" gorm:"uniqueIndex;size:64;default:''"`
	AppleUserID  string    `json:"apple_user_id" gorm:"uniqueIndex;size:128;default:''"`
	Gender       int8      `json:"gender" gorm:"default:0"` // 0=未知 1=男 2=女
	Birthday     string    `json:"birthday" gorm:"size:10;default:''"`
	Status       int8      `json:"status" gorm:"default:1"`  // 1=正常 2=冻结 3=注销
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

// Friendship 好友关系表
type Friendship struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     string    `json:"user_id" gorm:"index:idx_user_friend;size:32;not null"`
	FriendID   string    `json:"friend_id" gorm:"index:idx_user_friend;size:32;not null"`
	Status     int8      `json:"status" gorm:"default:1"` // 1=正常 2=已删除
	Source     string    `json:"source" gorm:"size:20;default:'search'"` // search/qrcode/recommend
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Friendship) TableName() string { return "friendships" }

// FriendRequest 好友申请表
type FriendRequest struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	FromUserID string    `json:"from_user_id" gorm:"index;size:32;not null"`
	ToUserID   string    `json:"to_user_id" gorm:"index;size:32;not null"`
	Message    string    `json:"message" gorm:"size:50;default:''"`
	Status     int8      `json:"status" gorm:"default:0"` // 0=待处理 1=已同意 2=已拒绝
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (FriendRequest) TableName() string { return "friend_requests" }

// Message 消息表
type Message struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	FromUserID   string    `json:"from_user_id" gorm:"index:idx_conversation;size:32;not null"`
	ToUserID     string    `json:"to_user_id" gorm:"index:idx_conversation;size:32;not null"`
	MsgType      int8      `json:"msg_type" gorm:"not null"` // 1=文本 2=图片 3=系统消息
	Content      string    `json:"content" gorm:"type:text"`
	IsRead       int8      `json:"is_read" gorm:"default:0"`   // 0=未读 1=已读
	IsBurn       int8      `json:"is_burn" gorm:"default:0"`   // 0=普通 1=阅后即焚
	BurnDuration int       `json:"burn_duration" gorm:"default:0"` // 焚毁倒计时秒数
	CreatedAt    time.Time `json:"created_at"`
}

func (Message) TableName() string { return "messages" }

// ConditionConfig 解锁条件配置表
type ConditionConfig struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     string    `json:"user_id" gorm:"index;size:32;not null"`
	CondType   int8      `json:"cond_type" gorm:"not null"`  // 1=定位 2=步数 3=答题
	IsEnabled  int8      `json:"is_enabled" gorm:"default:1"` // 1=启用 0=停用
	Params     string    `json:"params" gorm:"type:text"`     // JSON 参数
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (ConditionConfig) TableName() string { return "condition_configs" }

// UnlockRecord 解锁记录表
type UnlockRecord struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     string    `json:"user_id" gorm:"index;size:32;not null"`
	TargetID   string    `json:"target_id" gorm:"index;size:32;not null"` // 被解锁对象
	CondType   int8      `json:"cond_type" gorm:"not null"`
	Result     int8      `json:"result" gorm:"default:0"` // 0=验证中 1=成功 2=失败
	Detail     string    `json:"detail" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (UnlockRecord) TableName() string { return "unlock_records" }

// BurnRecord 阅后即焚记录表
type BurnRecord struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	MessageID int64     `json:"message_id" gorm:"index;not null"`
	UserID    string    `json:"user_id" gorm:"index;size:32;not null"`
	ReadAt    time.Time `json:"read_at"`    // 已读时间
	BurnAt    time.Time `json:"burn_at"`    // 焚毁时间
	Status    int8      `json:"status" gorm:"default:0"` // 0=未读 1=已读(倒计时) 2=已焚毁
	CreatedAt time.Time `json:"created_at"`
}

func (BurnRecord) TableName() string { return "burn_records" }

// QuizQuestion 答题题目
type QuizQuestion struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    string    `json:"user_id" gorm:"index;size:32;not null"`
	Question  string    `json:"question" gorm:"size:200;not null"`
	Answer    string    `json:"answer" gorm:"size:100;not null"`
	MaxTries  int       `json:"max_tries" gorm:"default:3"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (QuizQuestion) TableName() string { return "quiz_questions" }

// VerificationCode 验证码存储
type VerificationCode struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Phone     string    `json:"phone" gorm:"index;size:11;not null"`
	Code      string    `json:"code" gorm:"size:6"`
	CodeToken string    `json:"code_token" gorm:"index;size:64"`
	Type      string    `json:"type" gorm:"size:20;default:'code'"` // code | code_token
	ExpiresAt time.Time `json:"expires_at"`
	DailyCount int      `json:"daily_count" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
}

func (VerificationCode) TableName() string { return "verification_codes" }
