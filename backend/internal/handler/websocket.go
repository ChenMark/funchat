package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WSMessage WebSocket 消息格式
type WSMessage struct {
	Type      string      `json:"type"`       // chat | ping | pong | ack | read | typing
	From      string      `json:"from"`
	To        string      `json:"to"`
	MsgType   int8        `json:"msg_type"`   // 1=文本 2=图片 3=系统
	Content   string      `json:"content"`
	MsgID     int64       `json:"msg_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// Client WebSocket 客户端连接
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
}

// Hub WebSocket 连接管理器
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]*Client // userID -> Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *WSMessage
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有来源
	},
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *WSMessage, 256),
	}
}

// Run 启动 Hub（在 goroutine 中运行）
func (h *Hub) Run() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 如果已有旧连接，先关闭
			if old, exists := h.clients[client.UserID]; exists {
				close(old.Send)
				_ = old.Conn.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("[WS] 用户上线: %s (在线: %d)", client.UserID, h.OnlineCount())

		case client := <-h.unregister:
			h.mu.Lock()
			if existing, exists := h.clients[client.UserID]; exists && existing == client {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[WS] 用户下线: %s (在线: %d)", client.UserID, h.OnlineCount())

		case msg := <-h.broadcast:
			h.sendToUser(msg.To, msg)

		case <-ticker.C:
			// 定期清理超时连接
			h.mu.RLock()
			for _, client := range h.clients {
				_ = client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			}
			h.mu.RUnlock()
		}
	}
}

// OnlineCount 在线用户数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// IsOnline 用户是否在线
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[userID]
	return exists
}

// sendToUser 发送消息给指定用户
func (h *Hub) sendToUser(userID string, msg *WSMessage) {
	h.mu.RLock()
	client, exists := h.clients[userID]
	h.mu.RUnlock()

	if exists {
		data, _ := json.Marshal(msg)
		select {
		case client.Send <- data:
		default:
			// 发送缓冲区满，关闭连接
			h.mu.Lock()
			delete(h.clients, userID)
			close(client.Send)
			h.mu.Unlock()
			_ = client.Conn.Close()
		}
	}
	// 用户不在线时消息存储在数据库，上线后拉取离线消息
}

// SendToUser 外部调用：发送消息给指定用户
func (h *Hub) SendToUser(userID string, msg *WSMessage) {
	h.sendToUser(userID, msg)
}

// --- Client 读写 ---

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadLimit(65536)           // 64KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "ping":
			// 回复 pong
			pong := WSMessage{Type: "pong", Timestamp: time.Now().Unix()}
			data, _ := json.Marshal(pong)
			c.Send <- data

		case "chat":
			// Message persistence and authorization live on POST /messages/send.
			// Rejecting this legacy path prevents clients from receiving an ACK for
			// a message that was never durably stored.
			c.Send <- []byte(`{"type":"error","code":"use_rest_message_api"}`)
			continue
			// 聊天消息：存储 + 推送给接收方 + 回 ACK 给发送方
			msg.From = c.UserID
			msg.Timestamp = time.Now().Unix()

			// TODO: 存储到数据库
			// 模拟消息 ID
			msg.MsgID = time.Now().UnixNano()

			// 推送给接收方
			c.Hub.sendToUser(msg.To, &msg)

			// ACK 给发送方
			ack := WSMessage{Type: "ack", MsgID: msg.MsgID, Timestamp: time.Now().Unix()}
			ackData, _ := json.Marshal(ack)
			c.Send <- ackData

		case "typing":
			// 正在输入状态
			msg.From = c.UserID
			c.Hub.sendToUser(msg.To, &msg)

		case "read":
			// 已读回执
			msg.From = c.UserID
			c.Hub.sendToUser(msg.To, &msg)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// 心跳 Ping
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// --- WebSocket Handler ---

// WSHandler WebSocket 处理器
type WSHandler struct {
	Hub        *Hub
	JWTManager *jwt.Manager
}

// NewWSHandler 创建 WSHandler
func NewWSHandler(hub *Hub, jwtManager *jwt.Manager) *WSHandler {
	return &WSHandler{Hub: hub, JWTManager: jwtManager}
}

// HandleWS WebSocket 连接处理
// GET /ws?token=xxx
func (h *WSHandler) HandleWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Unauthorized(c, "缺少 token")
		return
	}

	claims, err := h.JWTManager.ParseToken(token)
	if err != nil {
		response.Unauthorized(c, "Token 无效")
		return
	}

	// 升级为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] 升级失败: %v", err)
		return
	}

	client := &Client{
		UserID: claims.UserID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h.Hub,
	}

	h.Hub.register <- client

	// 启动读写 goroutine
	go client.writePump()
	go client.readPump()

	log.Printf("[WS] 连接建立: %s", claims.UserID)
}

// OnlineStatus 查询在线状态
// GET /api/v1/online/:user_id
func (h *WSHandler) OnlineStatus(c *gin.Context) {
	userID := c.Param("user_id")
	online := h.Hub.IsOnline(userID)
	response.Success(c, gin.H{
		"user_id": userID,
		"online":  online,
	})
}
