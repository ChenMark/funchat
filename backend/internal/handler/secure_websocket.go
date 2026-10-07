package handler

import (
	"log"
	"net/http"
	"strings"

	"funchat/backend/pkg/jwt"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// SecureWSHandler keeps the WebSocket authentication and origin policy explicit.
// Mobile clients authenticate with the Authorization header during the handshake;
// tokens are never accepted from a URL query string.
type SecureWSHandler struct {
	hub            *Hub
	jwtManager     *jwt.Manager
	allowedOrigins map[string]struct{}
}

func NewSecureWSHandler(hub *Hub, jwtManager *jwt.Manager, allowedOrigins []string) *SecureWSHandler {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = struct{}{}
	}
	return &SecureWSHandler{hub: hub, jwtManager: jwtManager, allowedOrigins: origins}
}

func (h *SecureWSHandler) HandleWS(c *gin.Context) {
	token := bearerToken(c.GetHeader("Authorization"))
	if token == "" {
		response.Unauthorized(c, "missing bearer token")
		return
	}
	claims, err := h.jwtManager.ParseToken(token)
	if err != nil {
		response.Unauthorized(c, "invalid token")
		return
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" { // Native WebSocket clients do not send Origin.
			return true
		}
		_, ok := h.allowedOrigins[origin]
		return ok
	}}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] upgrade failed: %v", err)
		return
	}

	client := &Client{
		UserID: claims.UserID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
	}
	h.hub.register <- client
	go client.writePump()
	go client.readPump()
}

func bearerToken(value string) string {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
