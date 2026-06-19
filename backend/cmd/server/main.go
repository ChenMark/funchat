package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"funchat/backend/config"
	"funchat/backend/internal/handler"
	"funchat/backend/internal/middleware"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"
	"funchat/backend/pkg/sms"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 初始化数据库 + AutoMigrate
	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("[FATAL] 数据库连接失败: %v", err)
	}
	repo := repository.NewRepo(db)
	log.Println("[DB] Repo 初始化完成")

	// 初始化 JWT Manager
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
	)

	// 初始化腾讯云 SMS
	smsClient := sms.NewClient(
		cfg.SMS.SecretID, cfg.SMS.SecretKey,
		cfg.SMS.AppID, cfg.SMS.SignName, cfg.SMS.TemplateID,
	)
	log.Printf("[SMS] 客户端初始化完成 (已配置=%v)", smsClient.IsConfigured())

	// 初始化 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 全局中间件
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORSMiddleware())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API v1 路由组
	v1 := r.Group("/api/v1")

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(jwtManager, repo, smsClient)
	tpHandler := handler.NewThirdPartyHandler(jwtManager)
	friendHandler := handler.NewFriendHandler(repo)
	condHandler := handler.NewConditionHandler(repo)

	// WebSocket Hub
	hub := handler.NewHub()
	go hub.Run()
	wsHandler := handler.NewWSHandler(hub, jwtManager)
	msgHandler := handler.NewMessageHandler(hub, repo)
	burnHandler := handler.NewBurnHandler(repo)
	burnHandler.StartBurnCleanup(1 * time.Hour)

	// --- WebSocket 路由 (Token 通过 query 传递) ---
	r.GET("/ws", wsHandler.HandleWS)

	// --- 公开路由 (无需鉴权) ---
	auth := v1.Group("/auth")
	{
		auth.POST("/send-code", authHandler.SendCode)
		auth.POST("/verify-code", authHandler.VerifyCode)
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/wechat/login", tpHandler.WechatLogin)
		auth.POST("/wechat/bind", tpHandler.WechatBind)
		auth.POST("/apple/login", tpHandler.AppleLogin)
		auth.POST("/apple/bind", tpHandler.AppleBind)
	}

	// --- 鉴权路由 ---
	authorized := v1.Group("")
	authorized.Use(middleware.Auth(jwtManager))
	{
		authorized.POST("/auth/refresh", authHandler.RefreshToken)
		authorized.POST("/auth/logout", tpHandler.Logout)

		// 好友模块
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

		// 消息模块
		messages := authorized.Group("/messages")
		{
			messages.POST("/send", msgHandler.SendMessage)
			messages.POST("/send-conditional", condHandler.SendConditionalMessage)
			messages.POST("/upload-image", msgHandler.UploadImage)
			messages.POST("/upload-voice", msgHandler.UploadVoice)
			messages.GET("/history", msgHandler.GetHistory)
			messages.POST("/:id/read", msgHandler.MarkRead)
			messages.POST("/:id/revoke", condHandler.RevokeMessage)
			messages.GET("/:id/condition-status", condHandler.GetConditionStatus)
			// 阅后即焚模块
		messages.POST("/send-burn", burnHandler.SendBurnMessage)
		messages.POST("/:id/burn-read", burnHandler.ReadBurnMessage)
		messages.GET("/:id/burn-status", burnHandler.BurnStatus)
		messages.POST("/:id/burn-destroy", burnHandler.DestroyBurnMessage)
	}

		// 聊天列表
		authorized.GET("/conversations", msgHandler.GetConversations)

		// 在线状态
		authorized.GET("/online/:user_id", wsHandler.OnlineStatus)

		// 待销毁消息 (崩溃恢复)
		authorized.GET("/burns/pending", burnHandler.GetPendingBurns)

		// 解锁条件模块
		conditions := authorized.Group("/conditions")
		{
			conditions.POST("/set", condHandler.SetConditions)
			conditions.GET("/:user_id", condHandler.GetConditions)
			conditions.POST("/verify-location", condHandler.VerifyLocation)
			conditions.POST("/detect-fake-location", condHandler.DetectFakeLocation)
			conditions.POST("/verify-steps", condHandler.VerifySteps)
			conditions.POST("/detect-step-cheating", condHandler.DetectStepCheating)
			conditions.POST("/set-quiz", condHandler.SetQuiz)
			conditions.GET("/quiz/:id", condHandler.GetQuizQuestion)
			conditions.POST("/verify-quiz", condHandler.VerifyQuiz)
		}
	}

	// 启动服务
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 优雅退出
	go func() {
		log.Printf("🚀 FunChat 后端服务启动: http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务关闭异常: %v", err)
	}

	log.Println("服务已安全关闭")
}
