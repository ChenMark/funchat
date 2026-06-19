package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"funchat/backend/config"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// ==========================================
// Mock / Stub 打桩层
// ==========================================

// MockJWTManager 模拟 JWT Manager
func mockJWTManager() *jwt.Manager {
	return jwt.NewManager("test-secret", 7200, 604800)
}

// setupTestRouter 创建测试路由
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	jwtManager := mockJWTManager()

	// 测试用真实 MySQL
	cfg := config.DatabaseConfig{
		Host: "127.0.0.1", Port: "3306",
		User: "funchat", Password: "funchat123", DBName: "funchat",
	}
	db, _ := repository.NewDB(cfg)
	var repo *repository.Repo
	if db != nil {
		repo = repository.NewRepo(db)
	}

	authHandler := NewAuthHandler(jwtManager, repo, nil) // 测试用 nil smsClient
	tpHandler := NewThirdPartyHandler(jwtManager)
	friendHandler := NewFriendHandler(repo)
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond) // 等待 Hub 启动
	msgHandler := NewMessageHandler(hub, repo)
	condHandler := NewConditionHandler(repo)

	// 公开路由
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/send-code", authHandler.SendCode)
		auth.POST("/verify-code", authHandler.VerifyCode)
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/wechat/login", tpHandler.WechatLogin)
		auth.POST("/apple/login", tpHandler.AppleLogin)
	}

	// 鉴权路由
	authorized := r.Group("/api/v1")
	authorized.Use(func(c *gin.Context) {
		c.Set("user_id", "u_13800000001")
		c.Set("phone", "13800000001")
		c.Next()
	})
	{
		authorized.POST("/auth/refresh", authHandler.RefreshToken)

		friends := authorized.Group("/friends")
		{
			friends.GET("/search", friendHandler.Search)
			friends.POST("/request", friendHandler.SendRequest)
			friends.GET("/requests", friendHandler.GetRequests)
			friends.POST("/accept", friendHandler.AcceptRequest)
			friends.POST("/reject", friendHandler.RejectRequest)
			friends.GET("", friendHandler.ListFriends)
			friends.DELETE("/:id", friendHandler.DeleteFriend)
		}

		messages := authorized.Group("/messages")
		{
			messages.POST("/send", msgHandler.SendMessage)
			messages.POST("/send-conditional", condHandler.SendConditionalMessage)
			messages.GET("/history", msgHandler.GetHistory)
			messages.POST("/:id/revoke", condHandler.RevokeMessage)
		}

		conditions := authorized.Group("/conditions")
		{
			conditions.POST("/set", condHandler.SetConditions)
			conditions.GET("/:user_id", condHandler.GetConditions)
			conditions.POST("/verify-location", condHandler.VerifyLocation)
			conditions.POST("/detect-fake-location", condHandler.DetectFakeLocation)
			conditions.POST("/verify-steps", condHandler.VerifySteps)
			conditions.POST("/detect-step-cheating", condHandler.DetectStepCheating)
			conditions.POST("/set-quiz", condHandler.SetQuiz)
			conditions.POST("/verify-quiz", condHandler.VerifyQuiz)
		}
	}

	return r
}

// mockJSON 构建 JSON body
func mockJSON(data string) *strings.Reader {
	return strings.NewReader(data)
}

// parseResponse 解析测试响应
func parseResponse(t *testing.T, body string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("JSON 解析失败: %v, body: %s", err, body)
	}
	return result
}

// ==========================================
// Phase 1: 用户体系单元测试
// ==========================================

func TestAuthSendCode_Success(t *testing.T) {
	r := setupTestRouter()
	phone := fmt.Sprintf("138000%05d", time.Now().UnixNano()%100000)
	req, _ := http.NewRequest("POST", "/api/v1/auth/send-code",
		mockJSON(fmt.Sprintf(`{"phone":"%s"}`, phone)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("期望 200, 得到 %d: %s", w.Code, w.Body.String())
	}
	result := parseResponse(t, w.Body.String())
	if result["code"].(float64) != 0 {
		t.Errorf("期望 code=0, 得到 %v", result["code"])
	}
}

func TestAuthSendCode_InvalidPhone(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/send-code",
		mockJSON(`{"phone":"123"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	if result["code"].(float64) != 40000 {
		t.Errorf("期望 code=40000, 得到 %v", result["code"])
	}
}

func TestAuthSendCode_FrequencyLimit(t *testing.T) {
	r := setupTestRouter()
	phone := fmt.Sprintf("138999%05d", time.Now().UnixNano()%100000)

	// 第一次：成功
	req1, _ := http.NewRequest("POST", "/api/v1/auth/send-code",
		mockJSON(fmt.Sprintf(`{"phone":"%s"}`, phone)))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("第一次应成功, 得到 %d", w1.Code)
	}

	// 第二次：频控
	req2, _ := http.NewRequest("POST", "/api/v1/auth/send-code",
		mockJSON(fmt.Sprintf(`{"phone":"%s"}`, phone)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	result := parseResponse(t, w2.Body.String())
	if result["code"].(float64) != 40012 {
		t.Errorf("期望 code=40012 (频控), 得到 %v: %s", result["code"], w2.Body.String())
	}
}

func TestAuthVerifyCode_Invalid(t *testing.T) {
	r := setupTestRouter()
	phone := fmt.Sprintf("138777%05d", time.Now().UnixNano()%100000)

	// 先发验证码
	req1, _ := http.NewRequest("POST", "/api/v1/auth/send-code",
		mockJSON(fmt.Sprintf(`{"phone":"%s"}`, phone)))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	// 错误验证码
	req2, _ := http.NewRequest("POST", "/api/v1/auth/verify-code",
		mockJSON(fmt.Sprintf(`{"phone":"%s","code":"000000"}`, phone)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	result := parseResponse(t, w2.Body.String())
	if result["code"].(float64) != 40011 {
		t.Errorf("期望 code=40011 (验证码错误), 得到 %v", result["code"])
	}
}

func TestAuthWechatLogin(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/wechat/login",
		mockJSON(`{"code":"wx_test_code"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	if result["code"].(float64) != 0 {
		t.Errorf("期望 code=0, 得到 %v", result["code"])
	}
	data := result["data"].(map[string]interface{})
	if data["need_bind"] != true {
		t.Errorf("期望 need_bind=true, 得到 %v", data["need_bind"])
	}
}

func TestAuthAppleLogin(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/auth/apple/login",
		mockJSON(`{"identity_token":"apple_test_token"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	if result["code"].(float64) != 0 {
		t.Errorf("期望 code=0, 得到 %v: %s", result["code"], w.Body.String())
	}
}

// ==========================================
// Phase 1: 好友模块单元测试
// ==========================================

func TestFriendRequest_Self(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/friends/request",
		mockJSON(`{"target_user_id":"u_13800000001"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	if result["code"].(float64) != 40030 {
		t.Errorf("期望 code=40030 (不能添加自己), 得到 %v", result["code"])
	}
}

func TestFriendRequest_Duplicate(t *testing.T) {
	r := setupTestRouter()

	// 第一次申请
	req1, _ := http.NewRequest("POST", "/api/v1/friends/request",
		mockJSON(`{"target_user_id":"u_13900001111"}`))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Fatalf("第一次申请应成功, 得到 %d", w1.Code)
	}

	// 重复申请（code 是 40900 或 40911 均可接受）
	req2, _ := http.NewRequest("POST", "/api/v1/friends/request",
		mockJSON(`{"target_user_id":"u_13900001111"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	result := parseResponse(t, w2.Body.String())
	code := result["code"].(float64)
	if code != 40911 && code != 40900 {
		t.Errorf("期望重复申请拦截, 得到 code=%v", code)
	}
}

// ==========================================
// Phase 2: 消息模块单元测试
// ==========================================

func TestMessageSend(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/messages/send",
		mockJSON(`{"to_user_id":"u_13800000002","msg_type":1,"content":"Hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("期望 201, 得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestMessageHistory(t *testing.T) {
	r := setupTestRouter()

	// 先发一条消息
	req1, _ := http.NewRequest("POST", "/api/v1/messages/send",
		mockJSON(`{"to_user_id":"u_13800000002","msg_type":1,"content":"History Test"}`))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	// 查历史
	req2, _ := http.NewRequest("GET", "/api/v1/messages/history?friend_id=u_13800000002&page=1&size=20", nil)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != 200 {
		t.Errorf("期望 200, 得到 %d", w2.Code)
	}
}

// ==========================================
// Phase 3: 解锁条件单元测试
// ==========================================

func TestConditionSend(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/messages/send-conditional",
		mockJSON(`{"to_user_id":"u_13800000002","content":"条件消息","cond_types":[1,2,3]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("期望 201, 得到 %d: %s", w.Code, w.Body.String())
	}
}

func TestConditionSetAndGet(t *testing.T) {
	r := setupTestRouter()

	// 设置条件
	req1, _ := http.NewRequest("POST", "/api/v1/conditions/set",
		mockJSON(`{"cond_type":1,"is_enabled":1,"params":"{\"radius\":500}"}`))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Errorf("设置条件期望 201, 得到 %d", w1.Code)
	}

	// 获取条件
	req2, _ := http.NewRequest("GET", "/api/v1/conditions/u_13800000001", nil)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Errorf("获取条件期望 200, 得到 %d", w2.Code)
	}
}

func TestLocationVerify(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/conditions/verify-location",
		mockJSON(`{"target_user_id":"u_13800000002","lat":22.5431,"lng":113.9266}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("期望 200, 得到 %d", w.Code)
	}
	result := parseResponse(t, w.Body.String())
	data := result["data"].(map[string]interface{})
	if data["within_range"] != true {
		t.Errorf("期望 within_range=true, 得到 %v", data["within_range"])
	}
}

func TestFakeLocationDetect(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/conditions/detect-fake-location",
		mockJSON(`{"lat":22.5431,"lng":113.9266,"accuracy":15,"is_mock":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	data := result["data"].(map[string]interface{})
	if data["pass"] != true {
		t.Errorf("期望 pass=true, 得到 %v", data["pass"])
	}
}

func TestFakeLocationDetect_Mock(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/conditions/detect-fake-location",
		mockJSON(`{"lat":22.5431,"lng":113.9266,"accuracy":0,"is_mock":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	data := result["data"].(map[string]interface{})
	if data["suspicious"] != true {
		t.Errorf("期望 suspicious=true, 得到 %v", data["suspicious"])
	}
}

func TestStepsVerify(t *testing.T) {
	r := setupTestRouter()
	req, _ := http.NewRequest("POST", "/api/v1/conditions/verify-steps",
		mockJSON(`{"target_user_id":"u_13800000002","steps":9500}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	result := parseResponse(t, w.Body.String())
	data := result["data"].(map[string]interface{})
	if data["met"] != true {
		t.Errorf("期望 met=true, 得到 %v", data["met"])
	}
}

func TestQuizSetupAndVerify(t *testing.T) {
	r := setupTestRouter()

	// 设置题目
	req1, _ := http.NewRequest("POST", "/api/v1/conditions/set-quiz",
		mockJSON(`{"question":"Q?","answer":"A","max_tries":3}`))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != 201 {
		t.Fatalf("设置题目失败: %s", w1.Body.String())
	}

	// 正确答题
	req2, _ := http.NewRequest("POST", "/api/v1/conditions/verify-quiz",
		mockJSON(`{"quiz_id":1,"answer":"A"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	result := parseResponse(t, w2.Body.String())
	data := result["data"].(map[string]interface{})
	if data["correct"] != true {
		t.Errorf("期望 correct=true, 得到 %v", data["correct"])
	}
}

// ==========================================
// Benchmark 性能测试
// ==========================================

func BenchmarkAuthSendCode(b *testing.B) {
	r := setupTestRouter()
	for i := 0; i < b.N; i++ {
		// 每个测试用不同手机号避免频控
		phone := fmt.Sprintf("138%08d", i%100000000)
		body := fmt.Sprintf(`{"phone":"%s"}`, phone)
		req, _ := http.NewRequest("POST", "/api/v1/auth/send-code", mockJSON(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMessageSend(b *testing.B) {
	r := setupTestRouter()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/messages/send",
			mockJSON(`{"to_user_id":"u_13800000002","msg_type":1,"content":"bench"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
