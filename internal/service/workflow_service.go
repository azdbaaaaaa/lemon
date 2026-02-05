package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lemon/internal/model/workflow"
	wfrepo "lemon/internal/repository/workflow"
	"lemon/internal/pkg/id"
)

// NovelServiceForWorkflow 工作流服务需要的 NovelService 接口（避免循环导入）
type NovelServiceForWorkflow interface {
	CreateNovelFromResource(ctx context.Context, resourceID, userID, workflowID string) (string, error)
	SplitNovelIntoChapters(ctx context.Context, novelID string, targetChapters int) error
	GenerateNarrationsForAllChapters(ctx context.Context, novelID string) error
}

// WorkflowService 工作流服务接口
type WorkflowService interface {
	CreateWorkflow(ctx context.Context, userID string, req CreateWorkflowInput) (string, error)
	GetWorkflow(ctx context.Context, userID, workflowID string) (*workflow.Workflow, error)
	ListWorkflows(ctx context.Context, userID string, page, pageSize int64, status, stage string) (*WorkflowListResult, error)
	StartWorkflow(ctx context.Context, userID, workflowID string) error
}

// CreateWorkflowInput 创建工作流请求
type CreateWorkflowInput struct {
	Name          string `json:"name"`
	InputType     string `json:"input_type"`      // text 或 file
	ResourceID    string `json:"resource_id"`     // file 模式下必填
	TextContent   string `json:"text_content"`    // text 模式下必填
	ResourceSource string `json:"resource_source"` // existing/new（暂未使用，预留）
	NarrationType  string `json:"narration_type"` // 旁白类型：narration（旁白/解说）或 dialogue（真人对话）
}

// WorkflowListResult 工作流列表结果
type WorkflowListResult struct {
	Workflows []*workflow.Workflow
	Total     int64
	Page      int64
	PageSize  int64
}

type workflowService struct {
	repo            wfrepo.WorkflowRepository
	resourceService ResourceService
	novelService    NovelServiceForWorkflow // 添加 NovelService 依赖（使用接口避免循环导入）
}

// NewWorkflowService 创建 WorkflowService
func NewWorkflowService(repo wfrepo.WorkflowRepository, resourceService ResourceService, novelService NovelServiceForWorkflow) WorkflowService {
	return &workflowService{
		repo:            repo,
		resourceService: resourceService,
		novelService:    novelService,
	}
}

func (s *workflowService) CreateWorkflow(ctx context.Context, userID string, req CreateWorkflowInput) (string, error) {
	if req.Name == "" {
		return "", fmt.Errorf("name is required")
	}
	if req.InputType != "text" && req.InputType != "file" {
		return "", fmt.Errorf("invalid input_type, must be text or file")
	}
	if req.NarrationType != string(workflow.NarrationTypeNarration) && req.NarrationType != string(workflow.NarrationTypeDialogue) {
		return "", fmt.Errorf("invalid narration_type, must be narration or dialogue")
	}

	var resourceID string

	switch req.InputType {
	case "text":
		if req.TextContent == "" {
			return "", fmt.Errorf("text_content is required for input_type=text")
		}
		// 为 text 模式创建资源记录
		fileName := fmt.Sprintf("workflow_text_%s.txt", id.New())
		uploadReq := &UploadFileRequest{
			UserID:      userID,
			FileName:    fileName,
			ContentType: "text/plain; charset=utf-8",
			Ext:         "txt",
			Data:        strings.NewReader(req.TextContent),
		}
		uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
		if err != nil {
			return "", fmt.Errorf("failed to create resource for text content: %w", err)
		}
		resourceID = uploadResult.ResourceID
	case "file":
		if req.ResourceID == "" {
			return "", fmt.Errorf("resource_id is required for input_type=file")
		}
		// TODO: 验证资源是否存在且属于当前用户
		resourceID = req.ResourceID
	}

	wf := &workflow.Workflow{
		ID:            id.New(),
		UserID:        userID,
		Name:          req.Name,
		InputType:     req.InputType,
		ResourceID:    resourceID,
		TextContent:   req.TextContent, // 保留原始文本内容（可选）
		NarrationType: workflow.NarrationType(req.NarrationType),
		Status:        workflow.WorkflowStatusPending,
		CurrentStage:  "",
		Progress:      0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, wf); err != nil {
		return "", fmt.Errorf("create workflow: %w", err)
	}

	// TODO: 异步触发工作流处理（小说→章节→解说→音频→字幕→图片→视频）
	// 目前先留空，后续实现工作流编排逻辑
	// go s.processWorkflow(context.Background(), wf.ID)

	return wf.ID, nil
}

func (s *workflowService) GetWorkflow(ctx context.Context, userID, workflowID string) (*workflow.Workflow, error) {
	wf, err := s.repo.FindByID(ctx, workflowID, userID)
	if err != nil {
		return nil, fmt.Errorf("find workflow: %w", err)
	}
	return wf, nil
}

func (s *workflowService) ListWorkflows(ctx context.Context, userID string, page, pageSize int64, status, stage string) (*WorkflowListResult, error) {
	wfs, total, err := s.repo.ListByUser(ctx, userID, page, pageSize, status, stage)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	return &WorkflowListResult{
		Workflows: wfs,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// StartWorkflow 启动工作流处理
// 异步执行：创建小说 → 切分章节 → 生成解说
func (s *workflowService) StartWorkflow(ctx context.Context, userID, workflowID string) error {
	// 获取工作流信息
	wf, err := s.repo.FindByID(ctx, workflowID, userID)
	if err != nil {
		return fmt.Errorf("find workflow: %w", err)
	}

	// 检查工作流状态
	if wf.Status != workflow.WorkflowStatusPending {
		return fmt.Errorf("workflow status is %s, only pending workflows can be started", wf.Status)
	}

	// 更新工作流状态为 running，当前阶段为 script
	wf.Status = workflow.WorkflowStatusRunning
	wf.CurrentStage = workflow.WorkflowStageScript
	wf.Progress = 0.1 // 10% 进度（开始阶段）
	wf.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, wf); err != nil {
		return fmt.Errorf("update workflow status: %w", err)
	}

	// 异步执行工作流处理
	go s.processWorkflow(context.Background(), workflowID, userID, wf.ResourceID)

	return nil
}

// processWorkflow 异步处理工作流
// 步骤：1. 创建小说 2. 切分章节 3. 生成解说
func (s *workflowService) processWorkflow(ctx context.Context, workflowID, userID, resourceID string) {
	// 更新进度：创建小说阶段（10% - 30%）
	s.updateProgress(ctx, workflowID, 0.2, workflow.WorkflowStageScript)

	// 第一步：创建小说
	novelID, err := s.novelService.CreateNovelFromResource(ctx, resourceID, userID, workflowID)
	if err != nil {
		s.markWorkflowFailed(ctx, workflowID, fmt.Sprintf("创建小说失败: %v", err))
		return
	}

	s.updateProgress(ctx, workflowID, 0.3, workflow.WorkflowStageScript)

	// 第二步：切分章节（默认切分为 10 章）
	targetChapters := 10
	if err := s.novelService.SplitNovelIntoChapters(ctx, novelID, targetChapters); err != nil {
		s.markWorkflowFailed(ctx, workflowID, fmt.Sprintf("切分章节失败: %v", err))
		return
	}

	s.updateProgress(ctx, workflowID, 0.5, workflow.WorkflowStageScript)

	// 第三步：生成所有章节的解说
	if err := s.novelService.GenerateNarrationsForAllChapters(ctx, novelID); err != nil {
		s.markWorkflowFailed(ctx, workflowID, fmt.Sprintf("生成解说失败: %v", err))
		return
	}

	// 更新进度：完成 script 阶段（100%）
	s.updateProgress(ctx, workflowID, 1.0, workflow.WorkflowStageScript)

	// 工作流完成 script 阶段，等待用户继续后续步骤（asset, storyboard, animatic, video, edit）
	// 后续步骤可以通过前端手动触发或配置自动执行
}

// updateProgress 更新工作流进度
func (s *workflowService) updateProgress(ctx context.Context, workflowID string, progress float64, stage workflow.WorkflowStage) {
	wf, err := s.repo.FindByID(ctx, workflowID, "")
	if err != nil {
		return
	}

	wf.Progress = progress
	wf.CurrentStage = stage
	wf.UpdatedAt = time.Now()

	_ = s.repo.Update(ctx, wf)
}

// markWorkflowFailed 标记工作流为失败状态
func (s *workflowService) markWorkflowFailed(ctx context.Context, workflowID, errorMsg string) {
	wf, err := s.repo.FindByID(ctx, workflowID, "")
	if err != nil {
		return
	}

	wf.Status = workflow.WorkflowStatusFailed
	wf.UpdatedAt = time.Now()

	_ = s.repo.Update(ctx, wf)
}


