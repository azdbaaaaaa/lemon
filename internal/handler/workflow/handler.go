package workflow

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"lemon/internal/model/workflow"
	"lemon/internal/pkg/ctxutil"
	httputil "lemon/internal/pkg/http"
	"lemon/internal/service"
)

// ErrorResponse 复用通用错误响应
type ErrorResponse = httputil.ErrorResponse

// Handler 工作流处理器
type Handler struct {
	wfService service.WorkflowService
}

// NewHandler 创建工作流处理器
func NewHandler(wfService service.WorkflowService) *Handler {
	return &Handler{wfService: wfService}
}

// WorkflowInfo Workflow 列表 DTO
type WorkflowInfo struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	CurrentStage  string  `json:"current_stage"`
	Progress      float64 `json:"progress"`
	NarrationType string  `json:"narration_type"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	CompletedAt   string  `json:"completed_at,omitempty"`
}

// toWorkflowInfo 转换实体为 DTO
func toWorkflowInfo(w *workflow.Workflow) WorkflowInfo {
	info := WorkflowInfo{
		ID:            w.ID,
		Name:          w.Name,
		Status:        string(w.Status),
		CurrentStage:  string(w.CurrentStage),
		Progress:      w.Progress,
		NarrationType: string(w.NarrationType),
		CreatedAt:     w.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     w.UpdatedAt.Format(time.RFC3339),
	}
	if w.CompletedAt != nil {
		info.CompletedAt = w.CompletedAt.Format(time.RFC3339)
	}
	return info
}

// CreateWorkflowRequest 创建工作流请求
type CreateWorkflowRequest struct {
	Name          string `json:"name" binding:"required"`
	InputType     string `json:"input_type" binding:"required"` // text 或 file
	ResourceID    string `json:"resource_id,omitempty"`
	TextContent   string `json:"text_content,omitempty"`
	ResourceSource string `json:"resource_source,omitempty"`
	NarrationType  string `json:"narration_type" binding:"required"` // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
	// UserID 保留给兼容用途，正常情况下应从认证中间件注入的 context 中获取
	UserID string `json:"user_id,omitempty"`
}

// CreateWorkflowResponseData 创建工作流响应
type CreateWorkflowResponseData struct {
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

// CreateWorkflow 创建工作流
// @Summary      创建工作流
// @Description  创建新的工作流实例
// @Tags         工作流
// @Accept       json
// @Produce      json
// @Param        request  body      CreateWorkflowRequest  true  "创建工作流请求"
// @Success      201      {object}  map[string]interface{}  "成功响应"
// @Failure      400      {object}  ErrorResponse  "请求参数错误"
// @Failure      500      {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/workflow [post]
func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// 优先从 context 中解析 user_id（推荐方式）
	ctx := c.Request.Context()
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		// 兼容：如果 context 中没有，再尝试从请求体中读取（便于前期调试）
		userID = req.UserID
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    40101,
			Message: "unauthorized: user_id not found in context",
		})
		return
	}
	input := service.CreateWorkflowInput{
		Name:           req.Name,
		InputType:      req.InputType,
		ResourceID:     req.ResourceID,
		TextContent:    req.TextContent,
		ResourceSource: req.ResourceSource,
		NarrationType:  req.NarrationType,
	}

	id, err := h.wfService.CreateWorkflow(ctx, userID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40003,
			Message: err.Error(),
		})
		return
	}

	now := time.Now().Format(time.RFC3339)
	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data": CreateWorkflowResponseData{
			WorkflowID: id,
			Status:     string(workflow.WorkflowStatusPending),
			CreatedAt:  now,
		},
	})
}

// GetWorkflow 获取工作流详情
func (h *Handler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "id is required",
		})
		return
	}

	ctx := c.Request.Context()
	userID, _ := ctxutil.GetUserID(ctx)

	wf, err := h.wfService.GetWorkflow(ctx, userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    40401,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    toWorkflowInfo(wf),
	})
}

// ListWorkflowsResponseData 工作流列表响应
type ListWorkflowsResponseData struct {
	Workflows []WorkflowInfo `json:"workflows"`
	Total     int64          `json:"total"`
	Page      int64          `json:"page"`
	PageSize  int64          `json:"page_size"`
}

// ListWorkflows 查询工作流列表
func (h *Handler) ListWorkflows(c *gin.Context) {
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	status := c.Query("status")
	stage := c.Query("stage")

	page, _ := strconv.ParseInt(pageStr, 10, 64)
	pageSize, _ := strconv.ParseInt(pageSizeStr, 10, 64)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	ctx := c.Request.Context()
	userID, _ := ctxutil.GetUserID(ctx)

	result, err := h.wfService.ListWorkflows(ctx, userID, page, pageSize, status, stage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    50001,
			Message: err.Error(),
		})
		return
	}

	list := make([]WorkflowInfo, 0, len(result.Workflows))
	for _, wf := range result.Workflows {
		list = append(list, toWorkflowInfo(wf))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": ListWorkflowsResponseData{
			Workflows: list,
			Total:     result.Total,
			Page:      result.Page,
			PageSize:  result.PageSize,
		},
	})
}

// StartWorkflow 启动工作流
// @Summary      启动工作流
// @Description  启动待处理的工作流，开始执行剧本生成流程
// @Tags         工作流
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "工作流ID"
// @Success      200  {object}  map[string]interface{}  "成功响应"
// @Failure      400  {object}  ErrorResponse  "请求参数错误"
// @Failure      500  {object}  ErrorResponse  "服务器内部错误"
// @Router       /api/v1/workflow/{id}/start [post]
func (h *Handler) StartWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40001,
			Message: "id is required",
		})
		return
	}

	ctx := c.Request.Context()
	userID, ok := ctxutil.GetUserID(ctx)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Code:    40101,
			Message: "unauthorized: user_id not found in context",
		})
		return
	}

	if err := h.wfService.StartWorkflow(ctx, userID, id); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    40004,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "工作流已启动",
	})
}


