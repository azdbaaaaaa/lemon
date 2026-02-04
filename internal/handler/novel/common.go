package novel

import (
	"time"

	"lemon/internal/model/novel"
	httputil "lemon/internal/pkg/http"
)

// ErrorResponse 错误响应类型别名（使用共用的 http.ErrorResponse）
type ErrorResponse = httputil.ErrorResponse

// VideoInfo 视频信息（用于响应）
type VideoInfo struct {
	ID              string  `json:"id"`                // 视频ID
	ChapterID       string  `json:"chapter_id"`        // 章节ID
	NarrationID     string  `json:"narration_id"`      // 解说ID
	UserID          string  `json:"user_id"`           // 用户ID
	Sequence        int     `json:"sequence"`          // 序号
	VideoResourceID string  `json:"video_resource_id"` // 视频资源ID
	Duration        float64 `json:"duration"`          // 视频时长（秒）
	VideoType       string  `json:"video_type"`        // 视频类型：narration_video, final_video
	Prompt          string  `json:"prompt,omitempty"`  // 视频生成提示词
	Version         int     `json:"version"`           // 版本号
	Status          string  `json:"status"`            // 状态：pending, processing, completed, failed
	CreatedAt       string  `json:"created_at"`        // 创建时间
	UpdatedAt       string  `json:"updated_at"`        // 更新时间
}

// toVideoInfo 将Video实体转换为VideoInfo
func toVideoInfo(video *novel.Video) VideoInfo {
	return VideoInfo{
		ID:              video.ID,
		ChapterID:       video.ChapterID,
		NarrationID:     video.NarrationID,
		UserID:          video.UserID,
		Sequence:        video.Sequence,
		VideoResourceID: video.VideoResourceID,
		Duration:        video.Duration,
		VideoType:       string(video.VideoType),
		Prompt:          video.Prompt,
		Version:         video.Version,
		Status:          string(video.Status),
		CreatedAt:       video.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       video.UpdatedAt.Format(time.RFC3339),
	}
}

// toVideoInfoList 将Video列表转换为VideoInfo列表
func toVideoInfoList(videos []*novel.Video) []VideoInfo {
	result := make([]VideoInfo, len(videos))
	for i, video := range videos {
		result[i] = toVideoInfo(video)
	}
	return result
}

// NovelInfo 小说信息 DTO
type NovelInfo struct {
	ID          string `json:"id"`                    // 小说ID
	ResourceID  string `json:"resource_id"`           // 资源ID
	UserID      string `json:"user_id"`               // 用户ID
	WorkflowID  string `json:"workflow_id"`           // 工作流ID
	Title       string `json:"title,omitempty"`       // 小说名称
	Author      string `json:"author,omitempty"`      // 作者
	Description string `json:"description,omitempty"` // 简介
	CreatedAt   string `json:"created_at"`            // 创建时间
	UpdatedAt   string `json:"updated_at"`            // 更新时间
}

// toNovelInfo 将 Novel 实体转换为 NovelInfo DTO
func toNovelInfo(novelEntity *novel.Novel) NovelInfo {
	return NovelInfo{
		ID:          novelEntity.ID,
		ResourceID:  novelEntity.ResourceID,
		UserID:      novelEntity.UserID,
		WorkflowID:  novelEntity.WorkflowID,
		Title:       novelEntity.Title,
		Author:      novelEntity.Author,
		Description: novelEntity.Description,
		CreatedAt:   novelEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   novelEntity.UpdatedAt.Format(time.RFC3339),
	}
}

// ChapterInfo 章节信息 DTO
type ChapterInfo struct {
	ID          string `json:"id"`           // 章节ID
	NovelID     string `json:"novel_id"`     // 小说ID
	WorkflowID  string `json:"workflow_id"`  // 工作流ID
	UserID      string `json:"user_id"`      // 用户ID
	Sequence    int    `json:"sequence"`     // 章节序号
	Title       string `json:"title"`        // 章节标题
	ChapterText string `json:"chapter_text"` // 章节全文
	TotalChars  int    `json:"total_chars"`  // 章节总字符数
	WordCount   int    `json:"word_count"`   // 章节总字数
	LineCount   int    `json:"line_count"`   // 章节行数
	CreatedAt   string `json:"created_at"`   // 创建时间
	UpdatedAt   string `json:"updated_at"`   // 更新时间
}

// toChapterInfo 将 Chapter 实体转换为 ChapterInfo DTO
func toChapterInfo(chapterEntity *novel.Chapter) ChapterInfo {
	return ChapterInfo{
		ID:          chapterEntity.ID,
		NovelID:     chapterEntity.NovelID,
		WorkflowID:  chapterEntity.WorkflowID,
		UserID:      chapterEntity.UserID,
		Sequence:    chapterEntity.Sequence,
		Title:       chapterEntity.Title,
		ChapterText: chapterEntity.ChapterText,
		TotalChars:  chapterEntity.TotalChars,
		WordCount:   chapterEntity.WordCount,
		LineCount:   chapterEntity.LineCount,
		CreatedAt:   chapterEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   chapterEntity.UpdatedAt.Format(time.RFC3339),
	}
}

// toChapterInfoList 将 Chapter 实体列表转换为 ChapterInfo DTO 列表
func toChapterInfoList(chapters []*novel.Chapter) []ChapterInfo {
	list := make([]ChapterInfo, len(chapters))
	for i, chapter := range chapters {
		list[i] = toChapterInfo(chapter)
	}
	return list
}
