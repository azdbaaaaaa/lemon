package novel

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"lemon/internal/model/novel"
	novelrepo "lemon/internal/repository/novel"
)

// ContentService 内容生成服务接口
// 只包含生成内容的流程相关方法
type ContentService interface {
	// 一键生成内容（异步执行，立即返回）
	GenerateContent(ctx context.Context, novelID string, targetChapters int) error
	// 获取生成状态
	GetGenerationStatus(ctx context.Context, novelID string) (*GenerationStatusInfo, error)
}

// GenerationStatusInfo 生成状态信息
type GenerationStatusInfo struct {
	Status   novel.GenerationStatus `json:"status"`
	Progress int                    `json:"progress"` // 0-100
	Message  string                 `json:"message"`
}

// contentService 内容服务实现
// 通过组合 novelService 来实现 ContentService 接口
type contentService struct {
	novelService  NovelService
	novelRepo     novelrepo.NovelRepository
	chapterRepo   novelrepo.ChapterRepository
	sceneRepo     novelrepo.SceneRepository
	shotRepo      novelrepo.ShotRepository
	characterRepo novelrepo.CharacterRepository
	propRepo      novelrepo.PropRepository
}

// NewContentService 创建内容服务
func NewContentService(
	novelService NovelService,
	novelRepo novelrepo.NovelRepository,
	chapterRepo novelrepo.ChapterRepository,
	sceneRepo novelrepo.SceneRepository,
	shotRepo novelrepo.ShotRepository,
	characterRepo novelrepo.CharacterRepository,
	propRepo novelrepo.PropRepository,
) ContentService {
	return &contentService{
		novelService:  novelService,
		novelRepo:     novelRepo,
		chapterRepo:   chapterRepo,
		sceneRepo:     sceneRepo,
		shotRepo:      shotRepo,
		characterRepo: characterRepo,
		propRepo:      propRepo,
	}
}

// updateGenerationStatus 更新生成状态
func (s *contentService) updateGenerationStatus(ctx context.Context, novelID string, status novel.GenerationStatus, progress int, message string) error {
	updates := bson.M{
		"generation_status":   status,
		"generation_progress": progress,
		"generation_message":  message,
	}
	return s.novelRepo.Update(ctx, novelID, updates)
}

// GenerateContent 一键生成内容（异步执行，立即返回）
// 包括：切分章节、生成场景和镜头（包含角色和道具）
func (s *contentService) GenerateContent(ctx context.Context, novelID string, targetChapters int) error {
	// 检查小说是否存在
	novelEntity, err := s.novelService.GetNovel(ctx, novelID)
	if err != nil {
		return fmt.Errorf("novel not found: %w", err)
	}

	// 如果已经在生成中，返回错误
	if novelEntity.GenerationStatus == novel.GenerationStatusProcessing {
		return fmt.Errorf("generation is already in progress")
	}

	// 设置初始状态为 pending
	if err := s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusPending, 0, "准备开始生成..."); err != nil {
		return fmt.Errorf("failed to update generation status: %w", err)
	}

	// 启动异步任务
	go s.generateContentAsync(context.Background(), novelID, targetChapters)

	return nil
}

// clearExistingData 清除已有的数据
func (s *contentService) clearExistingData(ctx context.Context, novelID string) error {
	// 清除章节（会级联清除场景和镜头，因为它们关联chapter_id）
	if err := s.chapterRepo.DeleteByNovelID(ctx, novelID); err != nil {
		return fmt.Errorf("failed to delete chapters: %w", err)
	}

	// 清除场景
	if err := s.sceneRepo.DeleteByNovelID(ctx, novelID); err != nil {
		return fmt.Errorf("failed to delete scenes: %w", err)
	}

	// 清除镜头
	if err := s.shotRepo.DeleteByNovelID(ctx, novelID); err != nil {
		return fmt.Errorf("failed to delete shots: %w", err)
	}

	// 清除角色
	if err := s.characterRepo.DeleteByNovelID(ctx, novelID); err != nil {
		return fmt.Errorf("failed to delete characters: %w", err)
	}

	// 清除道具
	if err := s.propRepo.DeleteByNovelID(ctx, novelID); err != nil {
		return fmt.Errorf("failed to delete props: %w", err)
	}

	return nil
}

// generateContentAsync 异步执行内容生成
func (s *contentService) generateContentAsync(ctx context.Context, novelID string, targetChapters int) {
	// 设置状态为处理中
	_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusProcessing, 5, "开始清除已有数据...")

	// 步骤0: 清除已有数据
	if err := s.clearExistingData(ctx, novelID); err != nil {
		_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusFailed, 0, fmt.Sprintf("清除已有数据失败: %v", err))
		return
	}

	// 步骤1: 切分章节
	_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusProcessing, 10, "正在切分章节...")
	if err := s.novelService.SplitChapters(ctx, novelID, targetChapters); err != nil {
		_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusFailed, 0, fmt.Sprintf("切分章节失败: %v", err))
		return
	}

	// 步骤2: 获取章节列表
	_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusProcessing, 20, "章节切分完成，正在获取章节列表...")
	chapters, err := s.novelService.GetChapters(ctx, novelID)
	if err != nil {
		_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusFailed, 0, fmt.Sprintf("获取章节列表失败: %v", err))
		return
	}

	if len(chapters) == 0 {
		_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusFailed, 0, "切分章节后未找到章节")
		return
	}

	// 步骤3: 生成人物和道具描述（为每个章节生成）
	_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusProcessing, 30, "正在生成人物资源...")
	// 注意：CreateDefaultScenesAndShots 已经包含了生成和保存角色、道具的逻辑
	// 这里我们直接进入场景和镜头生成

	// 步骤4: 生成场景和镜头
	_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusProcessing, 50, "正在生成场景和镜头...")
	if err := s.novelService.CreateDefaultScenesAndShots(ctx, novelID, chapters[0].ID, chapters[0].UserID); err != nil {
		_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusFailed, 0, fmt.Sprintf("生成场景和镜头失败: %v", err))
		return
	}

	// 完成
	_ = s.updateGenerationStatus(ctx, novelID, novel.GenerationStatusCompleted, 100, "所有内容生成完成！")
}

// GetGenerationStatus 获取生成状态
func (s *contentService) GetGenerationStatus(ctx context.Context, novelID string) (*GenerationStatusInfo, error) {
	novelEntity, err := s.novelService.GetNovel(ctx, novelID)
	if err != nil {
		return nil, fmt.Errorf("novel not found: %w", err)
	}

	return &GenerationStatusInfo{
		Status:   novelEntity.GenerationStatus,
		Progress: novelEntity.GenerationProgress,
		Message:  novelEntity.GenerationMessage,
	}, nil
}
