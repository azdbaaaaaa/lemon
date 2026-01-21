package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lemon/internal/model"
	"lemon/internal/pkg/cache"
	"lemon/internal/pkg/mongodb"
	"lemon/internal/repository"
)

// ConversationHandler 对话管理处理器
type ConversationHandler struct {
	repo  *repository.ConversationRepo
	cache *cache.RedisCache
}

// NewConversationHandler 创建对话管理处理器
func NewConversationHandler(mongo *mongodb.Client, redisCache *cache.RedisCache) *ConversationHandler {
	var repo *repository.ConversationRepo
	if mongo != nil {
		repo = repository.NewConversationRepo(mongo.Database())
	}

	return &ConversationHandler{
		repo:  repo,
		cache: redisCache,
	}
}

// Create 创建对话
func (h *ConversationHandler) Create(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    50301,
			Message: "Database not available",
		})
		return
	}

	var req model.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	conv := &model.Conversation{
		UserID: req.UserID,
		Title:  req.Title,
		Model:  req.Model,
	}

	if err := h.repo.Create(c.Request.Context(), conv); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    50001,
			Message: "Failed to create conversation",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, conv)
}

// List 获取对话列表
func (h *ConversationHandler) List(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    50301,
			Message: "Database not available",
		})
		return
	}

	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Code:    40002,
			Message: "user_id is required",
		})
		return
	}

	convs, err := h.repo.ListByUserID(c.Request.Context(), userID, 20, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    50001,
			Message: "Failed to list conversations",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": convs,
		"total":         len(convs),
	})
}

// Get 获取对话详情
func (h *ConversationHandler) Get(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    50301,
			Message: "Database not available",
		})
		return
	}

	id := c.Param("id")

	conv, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Code:    40401,
			Message: "Conversation not found",
		})
		return
	}

	c.JSON(http.StatusOK, conv)
}

// Delete 删除对话
func (h *ConversationHandler) Delete(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Code:    50301,
			Message: "Database not available",
		})
		return
	}

	id := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Code:    50001,
			Message: "Failed to delete conversation",
			Detail:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Conversation deleted",
	})
}
