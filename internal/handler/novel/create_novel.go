package novel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateNovelRequest 创建小说请求
type CreateNovelRequest struct {
	ResourceID string `json:"resource_id" binding:"required"` // 资源ID（必填）
	UserID     string `json:"user_id" binding:"required"`     // 用户ID（必填）
	WorkflowID string `json:"workflow_id" binding:"required"` // 工作流ID（必填）
}

// CreateNovelResponseData 创建小说响应数据
type CreateNovelResponseData struct {
	NovelID string `json:"novel_id"` // 创建的小说ID
}

// CreateNovel 根据资源ID创建小说
// @Summary      创建小说
// @Description  根据资源ID创建小说，返回小说ID。这是小说处理流程的第一步。
// @Tags         小说管理
// @Accept       json
// @Produce      json
// @Param        request  body      CreateNovelRequest  true  "创建小说请求"
// @Success      201      {object}  map[string]interface{}  "成功响应"  "{\"code\": 0, \"message\": \"小说创建成功\", \"data\": {\"novel_id\": \"...\"}}"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/novels [post]
func (h *Handler) CreateNovel(c *gin.Context) {
	var req CreateNovelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 调用Service层
	novelID, err := h.novelService.CreateNovelFromResource(ctx, req.ResourceID, req.UserID, req.WorkflowID)
	if err != nil {
		code := http.StatusInternalServerError
		errorCode := 50001

		// 根据错误类型设置错误码
		if err.Error() == "failed to find resource" {
			code = http.StatusBadRequest
			errorCode = 40002
		}

		c.JSON(code, ErrorResponse{
			Code:    errorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "小说创建成功",
		"data": CreateNovelResponseData{
			NovelID: novelID,
		},
	})
}
