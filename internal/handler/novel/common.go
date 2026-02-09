package novel

import (
	"context"
	"time"

	"lemon/internal/model/novel"
	httputil "lemon/internal/pkg/http"
	"lemon/internal/service"
)

// ErrorResponse 错误响应类型别名（使用共用的 http.ErrorResponse）
type ErrorResponse = httputil.ErrorResponse

// VideoInfo 视频信息（用于响应）
type VideoInfo struct {
	ID              string  `json:"id"`                      // 视频ID
	ChapterID       string  `json:"chapter_id"`              // 章节ID
	NarrationID     string  `json:"narration_id,omitempty"`  // 解说ID
	ShotID          string  `json:"shot_id,omitempty"`       // 镜头ID
	UserID          string  `json:"user_id"`                 // 用户ID
	Sequence        int     `json:"sequence,omitempty"`      // 序号
	VideoResourceID string  `json:"video_resource_id"`       // 视频资源ID
	VideoURL        string  `json:"video_url,omitempty"`     // 视频直接访问URL
	Duration        float64 `json:"duration"`                // 视频时长（秒）
	VideoType       string  `json:"video_type"`              // 视频类型：narration_video, final_video, shot
	Prompt          string  `json:"prompt,omitempty"`        // 视频生成提示词
	Version         int     `json:"version"`                 // 版本号
	Status          string  `json:"status"`                  // 状态：pending, processing, completed, failed
	ErrorMessage    string  `json:"error_message,omitempty"` // 错误信息
	CreatedAt       string  `json:"created_at"`              // 创建时间
	UpdatedAt       string  `json:"updated_at"`              // 更新时间
}

// toVideoInfo 将Video实体转换为VideoInfo
func toVideoInfo(video *novel.Video, videoURL string) VideoInfo {
	return VideoInfo{
		ID:              video.ID,
		ChapterID:       video.ChapterID,
		NarrationID:     "", // 已删除，保留字段以兼容旧API
		ShotID:          video.ShotID,
		UserID:          video.UserID,
		Sequence:        0, // 已删除，保留字段以兼容旧API
		VideoResourceID: video.VideoResourceID,
		VideoURL:        videoURL,
		Duration:        video.Duration,
		VideoType:       string(video.VideoType),
		Prompt:          video.Prompt,
		Version:         video.Version,
		Status:          string(video.Status),
		ErrorMessage:    video.ErrorMessage,
		CreatedAt:       video.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       video.UpdatedAt.Format(time.RFC3339),
	}
}

// toVideoInfoList 将Video列表转换为VideoInfo列表
func toVideoInfoList(ctx context.Context, videos []*novel.Video, resourceService service.ResourceService) []VideoInfo {
	result := make([]VideoInfo, len(videos))
	for i, video := range videos {
		// 获取视频URL
		videoURL, _ := resourceService.GetDownloadURL(ctx, &service.GetDownloadURLRequest{
			ResourceID: video.VideoResourceID,
		})
		url := ""
		if videoURL != nil {
			url = videoURL.DownloadURL
		}
		result[i] = toVideoInfo(video, url)
	}
	return result
}
