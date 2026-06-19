package handler

import (
	"fmt"
	"net/http"

	"funchat/backend/internal/model"
	"funchat/backend/internal/repository"
	"funchat/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// FriendHandler 好友模块 handler
type FriendHandler struct {
	repo *repository.Repo
}

// NewFriendHandler 创建 FriendHandler
func NewFriendHandler(repo *repository.Repo) *FriendHandler {
	return &FriendHandler{repo: repo}
}

// Search 搜索用户
// GET /api/v1/friends/search?keyword=xxx
func (h *FriendHandler) Search(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		response.BadRequest(c, "请输入搜索关键词")
		return
	}

	currentUserID := c.GetString("user_id")

	users, err := h.repo.SearchUsers(keyword, currentUserID)
	if err != nil {
		response.Success(c, gin.H{"keyword": keyword, "users": []interface{}{}})
		return
	}

	response.Success(c, gin.H{
		"keyword": keyword,
		"users":   users,
	})
}

// SendRequest 发送好友申请
// POST /api/v1/friends/request
func (h *FriendHandler) SendRequest(c *gin.Context) {
	var req struct {
		TargetUserID string `json:"target_user_id" binding:"required"`
		Message      string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	currentUserID := c.GetString("user_id")

	requestID, err := h.repo.CreateFriendRequest(currentUserID, req.TargetUserID, req.Message)
	if err != nil {
		switch err.Error() {
		case "self":
		response.Error(c, http.StatusBadRequest, 40030, "不能添加自己为好友")
		case "already_friend":
		response.Conflict(c, "已经是好友了")
		case "duplicate":
		response.Conflict(c, "已发送过好友申请，请等待对方处理")
		default:
		response.InternalError(c)
		}
		return
	}

	response.Created(c, gin.H{
		"request_id": requestID,
		"message":    "好友申请已发送",
	})
}

// GetRequests 获取好友申请列表
// GET /api/v1/friends/requests
func (h *FriendHandler) GetRequests(c *gin.Context) {
	currentUserID := c.GetString("user_id")

	reqs, err := h.repo.GetPendingRequests(currentUserID)
	if err != nil {
		reqs = []model.FriendRequest{}
	}

	response.Success(c, gin.H{
		"requests": reqs,
	})
}

// AcceptRequest 同意好友申请
// POST /api/v1/friends/accept
func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	var req struct {
		RequestID int64 `json:"request_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	currentUserID := c.GetString("user_id")

	err := h.repo.AcceptFriendRequest(fmt.Sprintf("%d", req.RequestID), currentUserID)
	if err != nil {
		switch err.Error() {
		case "not_found":
		response.Error(c, http.StatusNotFound, 40410, "好友申请不存在或已处理")
		case "forbidden":
		response.Forbidden(c, "无权操作")
		default:
		response.InternalError(c)
		}
		return
	}

	response.Success(c, gin.H{
		"message":   "已添加为好友",
	})
}

// RejectRequest 拒绝好友申请
// POST /api/v1/friends/reject
func (h *FriendHandler) RejectRequest(c *gin.Context) {
	var req struct {
		RequestID int64 `json:"request_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	currentUserID := c.GetString("user_id")

	err := h.repo.RejectFriendRequest(fmt.Sprintf("%d", req.RequestID), currentUserID)
	if err != nil {
		switch err.Error() {
		case "not_found":
		response.Error(c, http.StatusNotFound, 40410, "好友申请不存在或已处理")
		case "forbidden":
		response.Forbidden(c, "无权操作")
		default:
		response.InternalError(c)
		}
		return
	}

	response.Success(c, gin.H{"message": "已拒绝"})
}

// ListFriends 获取好友列表
// GET /api/v1/friends
func (h *FriendHandler) ListFriends(c *gin.Context) {
	currentUserID := c.GetString("user_id")

	friendships, err := h.repo.GetFriends(currentUserID)
	if err != nil {
		friendships = []model.Friendship{}
	}

	var friends []gin.H
	for _, f := range friendships {
		friends = append(friends, gin.H{
			"user_id":   f.FriendID,
			"nickname":  fmt.Sprintf("好友_%s", f.FriendID[2:6]),
			"avatar":    "",
			"last_msg":  "最后一条消息...",
			"last_time": "10:30",
			"unread":    0,
		})
	}

	if friends == nil {
		friends = []gin.H{}
	}

	response.Success(c, gin.H{
		"friends": friends,
	})
}

// DeleteFriend 删除好友
// DELETE /api/v1/friends/:id
func (h *FriendHandler) DeleteFriend(c *gin.Context) {
	friendID := c.Param("id")
	currentUserID := c.GetString("user_id")

	err := h.repo.DeleteFriend(currentUserID, friendID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "已删除好友"})
}
