package novel

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/ffmpeg"
	"lemon/internal/pkg/id"
	"lemon/internal/service"
)

// VideoService 章节视频服务接口
// 定义章节视频相关的能力
type VideoService interface {
	// GenerateNarrationVideosForChapter 为章节生成所有 narration 视频（对应 concat_narration_video.py）
	// 合并 narration 视频，添加 BGM 和音效
	// 所有视频都使用图生视频方式（Ark API），不再需要 first_video
	GenerateNarrationVideosForChapter(ctx context.Context, chapterID string) ([]string, error)

	// GenerateVideosForChapter 为章节的所有 shot 生成视频
	// 基于章节的 active_scene_version 获取所有 shot，为每个 shot 生成视频
	GenerateVideosForChapter(ctx context.Context, chapterID string) ([]string, error)

	// GenerateFinalVideoForChapter 生成章节的最终完整视频
	// 注意：narration 模块已删除，此方法需要重构
	GenerateFinalVideoForChapter(ctx context.Context, chapterID string) (string, error)

	// GenerateFinalVideoForChapterWithVersion 指定视频版本号，手动确认后再合并生成最终视频
	// 注意：narration 模块已删除，此方法需要重构
	GenerateFinalVideoForChapterWithVersion(ctx context.Context, chapterID string, version int) (string, error)

	// GetVideoVersions 获取章节的所有视频版本号
	// 注意：narration 模块已删除，此方法需要重构
	GetVideoVersions(ctx context.Context, chapterID string) ([]int, error)

	// GetVideosByStatus 根据状态查询视频（用于轮询）
	GetVideosByStatus(ctx context.Context, status novel.VideoStatus) ([]*novel.Video, error)

	// ListVideosByChapter 获取章节视频列表（可指定版本；version<=0 则取最新版本）
	// 注意：narration 模块已删除，此方法需要重构
	ListVideosByChapter(ctx context.Context, chapterID string, version int) ([]*novel.Video, int, error)

	// GenerateVideoForShot 为单个镜头生成视频
	GenerateVideoForShot(ctx context.Context, shotID string) (string, error)

	// GetVideosByShot 获取镜头的所有视频
	GetVideosByShot(ctx context.Context, shotID string) ([]*novel.Video, error)
}

// GenerateFirstVideosForChapter 已废弃：现在所有视频都使用图生视频方式，不再需要 first_video
// DEPRECATED: 使用 GenerateNarrationVideosForChapter 即可，所有视频都通过图生视频生成
func (s *novelService) GenerateFirstVideosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	return nil, fmt.Errorf("GenerateFirstVideosForChapter is deprecated, use GenerateNarrationVideosForChapter instead")
}

// GenerateNarrationVideosForChapter 为章节生成所有 narration 视频
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GenerateNarrationVideosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	// TODO: 重构此方法，不再依赖 narration
	// 应该从章节直接查询场景和镜头
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// GenerateVideosForChapter 为章节的所有 shot 生成视频
// 基于章节的 active_scene_version 获取所有 shot，为每个 shot 生成视频
func (s *novelService) GenerateVideosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	if chapter.ActiveSceneVersion == 0 {
		return nil, fmt.Errorf("chapter has no active scene version")
	}

	// 2. 获取该版本的所有 shot
	shots, err := s.shotRepo.FindByChapterIDAndVersion(ctx, chapterID, chapter.ActiveSceneVersion)
		if err != nil {
		return nil, fmt.Errorf("find shots: %w", err)
	}

	if len(shots) == 0 {
		return nil, fmt.Errorf("no shots found for chapter %s, version %d", chapterID, chapter.ActiveSceneVersion)
	}

	// 3. 获取下一个视频版本号
	nextVersion, err := s.getNextVideoVersion(ctx, chapterID, chapter.ActiveSceneVersion)
		if err != nil {
		return nil, fmt.Errorf("get next video version: %w", err)
		}

	// 4. 先为每个 shot 创建视频记录（状态为 pending），以便前端可以轮询
	var videoIDs []string
		for _, shot := range shots {
		videoID := id.New()
		video := &novel.Video{
			ID:        videoID,
			NovelID:   chapter.NovelID,
			UserID:    chapter.UserID,
			VideoType: novel.VideoTypeShot,
			ShotID:    shot.ID,
			ChapterID: chapter.ID,
			Version:   nextVersion,
			Status:    novel.VideoStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := s.videoRepo.Create(ctx, video); err != nil {
			log.Error().
				Err(err).
				Str("shot_id", shot.ID).
				Str("chapter_id", chapterID).
				Msg("创建视频记录失败")
			continue
		}

		videoIDs = append(videoIDs, videoID)
	}

	// 5. 异步生成视频
	go func() {
		ctx := context.Background()
		for idx, shot := range shots {
			if idx >= len(videoIDs) {
				continue
			}
			videoID := videoIDs[idx]
			if err := s.generateVideoForShot(ctx, videoID, shot, chapter, nextVersion); err != nil {
				log.Error().
					Err(err).
					Str("shot_id", shot.ID).
					Str("video_id", videoID).
					Str("chapter_id", chapterID).
					Msg("生成视频失败")
				// 更新状态为失败
				if updateErr := s.videoRepo.UpdateStatus(ctx, videoID, novel.VideoStatusFailed, err.Error()); updateErr != nil {
					log.Error().
						Err(updateErr).
						Str("video_id", videoID).
						Msg("更新视频状态失败")
				}
				continue
			}
		}
	}()

	// 6. 立即返回 video IDs（异步执行）
	return videoIDs, nil
}

// generateVideoForShot 为单个 shot 生成视频
// videoID: 已创建的视频记录ID
func (s *novelService) generateVideoForShot(ctx context.Context, videoID string, shot *novel.Shot, chapter *novel.Chapter, version int) error {
	// 1. 更新状态为 processing
	if err := s.videoRepo.UpdateStatus(ctx, videoID, novel.VideoStatusProcessing, ""); err != nil {
		return fmt.Errorf("update video status to processing: %w", err)
	}

	// 2. 检查 shot 是否有 video_prompt
	if shot.VideoPrompt == "" {
		return fmt.Errorf("shot %s has no video_prompt", shot.ID)
	}

	// 3. 获取 shot 的首图（优先）或末图
	var image *novel.Image

	// 优先使用首图
	firstImages, err := s.imageRepo.FindByShotIDAndTypeAndVersion(ctx, shot.ID, novel.ImageTypeShotFirst, version)
	if err == nil && len(firstImages) > 0 {
		image = firstImages[0]
	} else {
		// 如果没有首图，使用末图
		lastImages, err := s.imageRepo.FindByShotIDAndTypeAndVersion(ctx, shot.ID, novel.ImageTypeShotLast, version)
		if err == nil && len(lastImages) > 0 {
			image = lastImages[0]
		}
	}

	if image == nil {
		return fmt.Errorf("shot %s has no image (first or last)", shot.ID)
	}

	// 4. 下载图片
	downloadReq := service.DownloadFileRequest{
		ResourceID: image.ImageResourceID,
	}
	downloadResult, err := s.resourceService.DownloadFile(ctx, &downloadReq)
	if err != nil {
		return fmt.Errorf("download image: %w", err)
	}

	// 5. 读取图片数据并转换为 base64 data URL
	imageData, err := io.ReadAll(downloadResult.Data)
	if err != nil {
		return fmt.Errorf("read image data: %w", err)
	}
	defer downloadResult.Data.Close()

	// 转换为 base64 data URL
	imageDataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(imageData))

	// 6. 使用 VideoProvider 生成视频
	// 视频时长：根据 shot.Duration，最大 12 秒
	duration := int(shot.Duration)
	if duration <= 0 {
		duration = 5 // 默认 5 秒
	}
	if duration > 12 {
		duration = 12 // 最大 12 秒
	}

	videoData, err := s.videoProvider.GenerateVideoFromImage(ctx, imageDataURL, duration, shot.VideoPrompt)
	if err != nil {
		return fmt.Errorf("generate video: %w", err)
	}

	// 7. 上传视频文件到资源服务
	uploadReq := service.UploadFileRequest{
		FileName:    fmt.Sprintf("video_%s.mp4", id.New()),
		ContentType: "video/mp4",
		Ext:         "mp4",
		Data:        bytes.NewReader(videoData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, &uploadReq)
	if err != nil {
		return fmt.Errorf("upload video: %w", err)
	}

	// 8. 更新视频记录（设置 resource_id、duration、prompt 和状态）
	if err := s.videoRepo.UpdateVideoResourceID(ctx, videoID, uploadResult.ResourceID, float64(duration), shot.VideoPrompt); err != nil {
		return fmt.Errorf("update video resource: %w", err)
	}

	// 9. 更新状态为 completed
	if err := s.videoRepo.UpdateStatus(ctx, videoID, novel.VideoStatusCompleted, ""); err != nil {
		return fmt.Errorf("update video status to completed: %w", err)
	}

	log.Info().
		Str("shot_id", shot.ID).
		Str("video_id", videoID).
		Str("chapter_id", chapter.ID).
		Float64("duration", float64(duration)).
		Msg("视频生成成功")

	return nil
}

// generateNarrationVideosForChapter_old 旧版本（已废弃）
func (s *novelService) generateNarrationVideosForChapter_old(ctx context.Context, chapterID string) ([]string, error) {
	// 1. 从章节直接查询场景
	scenes, err := s.sceneRepo.FindByChapterID(ctx, chapterID)
			if err != nil {
		return nil, fmt.Errorf("find scenes: %w", err)
	}

	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes found for chapter")
	}

	// 2. 从 Scenes 和 Shots 中提取所有 Shots，按照顺序编号
	var allShots []struct {
		SceneSequence int
		ShotSequence  int
		Shot          *novel.Shot
		Index         int // 编号（从1开始）
	}

	shotIndex := 1
	for _, scene := range scenes {
		// 查询该场景下的所有镜头
		shots, err := s.shotRepo.FindBySceneID(ctx, scene.ID)
	if err != nil {
			continue
		}

		for _, shot := range shots {
			allShots = append(allShots, struct {
				SceneSequence int
				ShotSequence  int
				Shot          *novel.Shot
				Index         int
			}{
				SceneSequence: scene.Sequence,
				ShotSequence:  shot.Sequence,
				Shot:          shot,
				Index:         shotIndex,
			})
			shotIndex++
		}
	}

	if len(allShots) == 0 {
		return nil, fmt.Errorf("no shots found")
	}
	// 后续代码已废弃
	return nil, fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// generateNarration01Video 已废弃
func (s *novelService) generateNarration01Video(
	ctx context.Context,
	narration interface{}, // 已废弃
	video1 *novel.Video,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// replaceVideoAudio 替换视频的音频
func (s *novelService) replaceVideoAudio(ctx context.Context, videoPath, audioPath, outputPath string, ffmpegClient *ffmpeg.Client) error {
	// 使用 FFmpeg 替换音频
	// ffmpeg -i video.mp4 -i audio.mp3 -c:v copy -c:a aac -map 0:v:0 -map 1:a:0 output.mp4
	// 这里简化实现，直接使用 FFmpeg 命令
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-map", "0:v:0",
		"-map", "1:a:0",
		outputPath,
	)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg replace audio failed: %w", err)
	}

	return nil
}

// generateMergedNarrationVideo 生成合并的视频（内部实现：前3个场景合并）
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) generateMergedNarrationVideo(
	ctx context.Context,
	chapterID string,
	narration interface{}, // 已废弃：narration 模块已删除
	shots []struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.Shot
		Index       int
	},
	video1 *novel.Video, // 保留参数以保持接口兼容，但不再使用
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// generateMergedNarrationVideo_old 旧版本（已废弃）
func (s *novelService) generateMergedNarrationVideo_old(
	ctx context.Context,
	chapterID string,
	narration interface{},
	shots []struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.Shot
		Index       int
	},
	video1 *novel.Video,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	if len(shots) != 3 {
		return "", fmt.Errorf("merged video requires exactly 3 shots, got %d", len(shots))
	}

	// TODO: 重构此方法，不再依赖 narration
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// generateMergedNarrationVideo_old 旧版本（已废弃，代码已删除）
// 此方法依赖 narration 模块，已标记为废弃

// generateSingleNarrationVideo 生成单个场景的视频
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) generateSingleNarrationVideo(
	ctx context.Context,
	chapterID string,
	narration interface{}, // 已废弃：narration 模块已删除
	shotInfo struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.Shot
		Index       int
	},
	narrationNum string,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

// generateSingleNarrationVideo_old 旧版本（已废弃，代码已删除）
// 此方法依赖 narration 模块，已标记为废弃

// mergeAudioFiles 合并多个音频文件
func (s *novelService) mergeAudioFiles(ctx context.Context, audioPaths []string, outputPath string) error {
	// 使用 FFmpeg 合并音频
	// ffmpeg -i audio1.mp3 -i audio2.mp3 -i audio3.mp3 -filter_complex "[0:a][1:a][2:a]concat=n=3:v=0:a=1[out]" -map "[out]" output.mp3
	args := []string{"-y"}
	for _, audioPath := range audioPaths {
		args = append(args, "-i", audioPath)
	}

	// 构建 filter_complex
	filterComplex := ""
	for i := range audioPaths {
		if i > 0 {
			filterComplex += ";"
		}
		filterComplex += fmt.Sprintf("[%d:a]", i)
	}
	filterComplex += fmt.Sprintf("concat=n=%d:v=0:a=1[out]", len(audioPaths))

	args = append(args, "-filter_complex", filterComplex, "-map", "[out]", "-c:a", "aac", "-b:a", "128k", outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge audio failed: %w", err)
	}

	return nil
}

// mergeASSFiles 合并多个 ASS 字幕文件
// 合并策略：保留第一个文件的头部信息，合并所有文件的 Dialogue 事件，并调整时间戳
func (s *novelService) mergeASSFiles(ctx context.Context, assPaths []string, outputPath string) error {
	if len(assPaths) == 0 {
		return fmt.Errorf("no ASS files to merge")
	}

	// 读取第一个文件作为基础（包含头部信息）
	firstContent, err := os.ReadFile(assPaths[0])
	if err != nil {
		return fmt.Errorf("read first ASS file: %w", err)
	}

	// 解析第一个文件，提取头部和 Dialogue 事件
	firstLines := strings.Split(string(firstContent), "\n")
	var headerLines []string
	var firstDialogues []string
	inEventsSection := false
	for _, line := range firstLines {
		if strings.HasPrefix(line, "[Events]") {
			inEventsSection = true
			headerLines = append(headerLines, line)
			continue
		}
		if strings.HasPrefix(line, "Format:") {
			headerLines = append(headerLines, line)
			continue
		}
		if !inEventsSection {
			headerLines = append(headerLines, line)
		} else if strings.HasPrefix(line, "Dialogue:") {
			firstDialogues = append(firstDialogues, line)
		}
	}

	// 读取并解析其他文件的 Dialogue 事件
	allDialogues := firstDialogues
	var timeOffset float64

	// 计算第一个文件的时长（从最后一个 Dialogue 事件的结束时间）
	if len(firstDialogues) > 0 {
		lastDialogue := firstDialogues[len(firstDialogues)-1]
		// 解析时间戳：Dialogue: 0,Start,End,...
		parts := strings.Split(lastDialogue, ",")
		if len(parts) >= 3 {
			// 解析结束时间
			if endTime, err := parseASSTime(parts[2]); err == nil {
				timeOffset = endTime
			}
		}
	}

	// 读取其他文件
	for i := 1; i < len(assPaths); i++ {
		content, err := os.ReadFile(assPaths[i])
		if err != nil {
			return fmt.Errorf("read ASS file %d: %w", i+1, err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "[Events]") {
				continue
			}
			if strings.HasPrefix(line, "Dialogue:") {
				// 调整时间戳：添加时间偏移
				adjustedLine := adjustDialogueTime(line, timeOffset)
				allDialogues = append(allDialogues, adjustedLine)
			}
		}

		// 更新时间偏移（累加当前文件的时长）
		if len(allDialogues) > 0 {
			lastDialogue := allDialogues[len(allDialogues)-1]
			parts := strings.Split(lastDialogue, ",")
			if len(parts) >= 3 {
				if endTime, err := parseASSTime(parts[2]); err == nil {
					timeOffset = endTime
				}
			}
		}
	}

	// 合并头部和所有 Dialogue 事件
	mergedContent := strings.Join(headerLines, "\n")
	if !strings.HasSuffix(mergedContent, "\n") {
		mergedContent += "\n"
	}
	mergedContent += strings.Join(allDialogues, "\n") + "\n"

	// 写入输出文件
	if err := os.WriteFile(outputPath, []byte(mergedContent), 0644); err != nil {
		return fmt.Errorf("write merged ASS file: %w", err)
	}

	return nil
}

// parseASSTime 解析 ASS 时间格式 (H:MM:SS:CC) 转换为秒数
func parseASSTime(timeStr string) (float64, error) {
	timeStr = strings.TrimSpace(timeStr)
	lastColonIndex := strings.LastIndex(timeStr, ":")
	if lastColonIndex > 0 {
		timeStr = timeStr[:lastColonIndex] + "." + timeStr[lastColonIndex+1:]
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format: expected H:MM:SS.CC, got %s", timeStr)
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	secondsParts := strings.Split(parts[2], ".")
	if len(secondsParts) != 2 {
		return 0, fmt.Errorf("invalid time format: expected SS.CC, got %s", parts[2])
	}

	seconds, _ := strconv.Atoi(secondsParts[0])
	centiseconds, _ := strconv.Atoi(secondsParts[1])

	totalSeconds := float64(hours*3600 + minutes*60 + seconds)
	totalSeconds += float64(centiseconds) / 100.0

	return totalSeconds, nil
}

// formatTimeForASS 将秒数转换为 ASS 时间格式 (H:MM:SS:CC)
func formatTimeForASS(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((int(seconds) % 3600) / 60)
	secs := seconds - float64(hours*3600) - float64(minutes*60)
	centiseconds := int((secs - float64(int(secs))) * 100)
	return fmt.Sprintf("%d:%02d:%02d:%02d", hours, minutes, int(secs), centiseconds)
}

// adjustDialogueTime 调整 Dialogue 事件的时间戳（添加时间偏移）
func adjustDialogueTime(dialogueLine string, timeOffset float64) string {
	// Dialogue: 0,Start,End,Style,Name,MarginL,MarginR,MarginV,Effect,Text
	parts := strings.SplitN(dialogueLine, ",", 4)
	if len(parts) < 4 {
		return dialogueLine // 格式错误，返回原样
	}

	startTimeStr := parts[1]
	endTimeStr := parts[2]
	rest := parts[3]

	// 解析并调整时间戳
	startTime, err1 := parseASSTime(startTimeStr)
	endTime, err2 := parseASSTime(endTimeStr)

	if err1 != nil || err2 != nil {
		return dialogueLine // 解析失败，返回原样
	}

	// 调整时间戳
	newStartTime := startTime + timeOffset
	newEndTime := endTime + timeOffset

	// 格式化新的时间戳
	newStartTimeStr := formatTimeForASS(newStartTime)
	newEndTimeStr := formatTimeForASS(newEndTime)

	// 重新构建 Dialogue 行
	return fmt.Sprintf("Dialogue: %s,%s,%s,%s", parts[0], newStartTimeStr, newEndTimeStr, rest)
}

// GenerateFinalVideoForChapter 生成章节的最终完整视频
// 对应 Python: concat_finish_video.py
func (s *novelService) GenerateFinalVideoForChapter(ctx context.Context, chapterID string) (string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

func (s *novelService) GenerateFinalVideoForChapterWithVersion(ctx context.Context, chapterID string, version int) (string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

func (s *novelService) generateFinalVideoForChapter(ctx context.Context, chapterID string, version int) (string, error) {
	// TODO: 重构此方法，不再依赖 narration
	return "", fmt.Errorf("narration module has been removed, this method needs to be refactored")
}

func (s *novelService) generateFinalVideoForChapter_old(ctx context.Context, chapterID string, version int) (string, error) {
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	// 2. 确定要合并的版本号：version<=0 则取最新版本
	// TODO: 重构此方法，不再依赖 narration
	videoVersion := version
	if videoVersion <= 0 {
		videoVersion = 1 // 暂时返回1
	}

	// 2.5. 只获取指定版本的分镜视频
	// TODO: 重构此方法，使用新的查询方式
	narrationVideos, err := s.videoRepo.FindByShotID(ctx, "") // 需要重构
	if err != nil {
		return "", fmt.Errorf("find videos: %w", err)
	}

	// 过滤出分镜视频类型的视频
	var filteredNarrationVideos []*novel.Video
	for _, video := range narrationVideos {
		if video.VideoType == novel.VideoTypeShot {
			filteredNarrationVideos = append(filteredNarrationVideos, video)
		}
	}

	if len(filteredNarrationVideos) == 0 {
		return "", fmt.Errorf("no shot videos found for chapter %s, version %d", chapterID, videoVersion)
	}

	// 按创建时间排序（Video 模型已经没有 Sequence 字段）
	sort.Slice(filteredNarrationVideos, func(i, j int) bool {
		return filteredNarrationVideos[i].CreatedAt.Before(filteredNarrationVideos[j].CreatedAt)
	})

	narrationVideos = filteredNarrationVideos

	log.Info().
		Str("chapter_id", chapterID).
		Int("version", videoVersion).
		Int("narration_video_count", len(narrationVideos)).
		Msg("使用指定版本的 narration 视频进行合并")

	// 3. 初始化 FFmpeg 客户端
	ffmpegClient := ffmpeg.NewClient()

	// 4. 下载所有视频到临时文件
	tmpDir := os.TempDir()
	var videoPaths []string
	for idx, video := range narrationVideos {
		downloadReq := &service.DownloadFileRequest{
			ResourceID: video.VideoResourceID,
			UserID:     chapter.UserID,
		}
		videoResult, err := s.resourceService.DownloadFile(ctx, downloadReq)
		if err != nil {
			return "", fmt.Errorf("download video %d: %w", idx+1, err)
		}
		defer videoResult.Data.Close()

		tmpVideoPath := filepath.Join(tmpDir, fmt.Sprintf("video_%d_%s.mp4", idx+1, id.New()))
		defer os.Remove(tmpVideoPath)

		videoFile, err := os.Create(tmpVideoPath)
		if err != nil {
			return "", fmt.Errorf("create temp video file: %w", err)
		}
		if _, err := io.Copy(videoFile, videoResult.Data); err != nil {
			videoFile.Close()
			return "", fmt.Errorf("copy video data: %w", err)
		}
		videoFile.Close()

		videoPaths = append(videoPaths, tmpVideoPath)
	}

	// 5. 合并所有视频片段
	tmpMergedPath := filepath.Join(tmpDir, fmt.Sprintf("merged_%s.mp4", id.New()))
	defer os.Remove(tmpMergedPath)

	if err := ffmpegClient.ConcatVideos(ctx, videoPaths, tmpMergedPath); err != nil {
		return "", fmt.Errorf("concat videos: %w", err)
	}

	// 6. 添加 finish.mp4（如果存在）
	finishVideoPath := s.getFinishVideoPath()
	var finalVideoPath string
	if finishVideoPath != "" {
		// 检查 finish.mp4 是否存在
		if _, err := os.Stat(finishVideoPath); err == nil {
			// 创建包含 finish.mp4 的合并列表
			concatListPath := filepath.Join(tmpDir, fmt.Sprintf("final_concat_list_%s.txt", id.New()))
			defer os.Remove(concatListPath)

			// 写入视频文件列表
			concatListContent := fmt.Sprintf("file '%s'\nfile '%s'\n", tmpMergedPath, finishVideoPath)
			if err := os.WriteFile(concatListPath, []byte(concatListContent), 0644); err != nil {
				return "", fmt.Errorf("write concat list: %w", err)
			}

			// 使用 FFmpeg 拼接（使用流复制避免重新编码）
			tmpWithFinishPath := filepath.Join(tmpDir, fmt.Sprintf("with_finish_%s.mp4", id.New()))
			defer os.Remove(tmpWithFinishPath)

			args := []string{
				"-y",
				"-f", "concat",
				"-safe", "0",
				"-i", concatListPath,
				"-c", "copy", // 使用流复制而不是重新编码
				"-avoid_negative_ts", "make_zero", // 处理时间戳问题
				tmpWithFinishPath,
			}

			cmd := exec.CommandContext(ctx, "ffmpeg", args...)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("concat with finish video: %w, stderr: %s", err, stderr.String())
			}

			finalVideoPath = tmpWithFinishPath
		} else {
			log.Warn().Str("path", finishVideoPath).Msg("finish.mp4 文件不存在，跳过 finish 视频拼接")
			finalVideoPath = tmpMergedPath
		}
	} else {
		finalVideoPath = tmpMergedPath
	}

	// 7. 标准化视频分辨率
	tmpFinalPath := filepath.Join(tmpDir, fmt.Sprintf("final_%s.mp4", id.New()))
	defer os.Remove(tmpFinalPath)

	if err := ffmpegClient.StandardizeVideo(ctx, finalVideoPath, tmpFinalPath, 720, 1280, 30); err != nil {
		return "", fmt.Errorf("standardize video: %w", err)
	}

	// 8. 上传最终视频到 resource 模块
	finalVideoFile, err := os.Open(tmpFinalPath)
	if err != nil {
		return "", fmt.Errorf("open final video: %w", err)
	}
	defer finalVideoFile.Close()

	fileName := fmt.Sprintf("%s_final_video.mp4", chapterID)
	uploadReq := &service.UploadFileRequest{
		UserID:      chapter.UserID,
		FileName:    fileName,
		ContentType: "video/mp4",
		Ext:         "mp4",
		Data:        finalVideoFile,
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload video: %w", err)
	}

	// 9. 计算总时长
	var totalDuration float64
	for _, video := range narrationVideos {
		totalDuration += video.Duration
	}

	// 10. 创建最终视频记录
	// 使用与 narration 视频相同的版本号（已在前面获取）
	videoID := id.New()
	videoEntity := &novel.Video{
		ID:        videoID,
		ChapterID: chapterID,
		NovelID:   chapter.NovelID,
		UserID:    chapter.UserID,
		// Sequence removed
		VideoResourceID: uploadResult.ResourceID,
		Duration:        totalDuration,
		VideoType:       novel.VideoTypeNovel,
		Version:         videoVersion, // 使用与 narration 视频相同的版本号
		Status:          novel.VideoStatusCompleted,
	}

	if err := s.videoRepo.Create(ctx, videoEntity); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	return videoID, nil
}

// GetVideoVersions 获取章节的所有视频版本号
// 注意：narration 模块已删除，此方法需要重构
func (s *novelService) GetVideoVersions(ctx context.Context, chapterID string) ([]int, error) {
	// TODO: 重构此方法，根据章节ID查询视频版本号
	// 暂时返回空列表
	return []int{1}, nil
}

// GetVideosByStatus 根据状态查询视频（用于轮询）
func (s *novelService) GetVideosByStatus(ctx context.Context, status novel.VideoStatus) ([]*novel.Video, error) {
	return s.videoRepo.FindByStatus(ctx, status)
}

// getNextVideoVersion 获取章节的下一个视频版本号（自动递增）
func (s *novelService) getNextVideoVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	// 查询该章节已有的视频版本号
	videos, err := s.videoRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		// 如果查询失败，返回基础版本号
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	// 找到该章节的最大版本号
	maxVersion := 0
	for _, video := range videos {
		if video.Version > maxVersion {
			maxVersion = video.Version
		}
	}

	// 如果指定了基础版本号，使用它；否则使用最大版本号+1
	if baseVersion > 0 {
		if baseVersion > maxVersion {
			return baseVersion, nil
				}
				return maxVersion + 1, nil
	}

	if maxVersion == 0 {
		return 1, nil
	}
	return maxVersion + 1, nil
}

// GenerateVideoForShot 为单个镜头生成视频
func (s *novelService) GenerateVideoForShot(ctx context.Context, shotID string) (string, error) {
	// 1. 获取镜头信息
	shot, err := s.shotRepo.FindByID(ctx, shotID)
	if err != nil {
		return "", fmt.Errorf("find shot: %w", err)
	}

	// 2. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, shot.ChapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	// 3. 获取下一个视频版本号
	nextVersion, err := s.getNextVideoVersion(ctx, shot.ChapterID, shot.Version)
	if err != nil {
		return "", fmt.Errorf("get next video version: %w", err)
	}

	// 4. 先创建视频记录（状态为 pending）
	videoID := id.New()
	video := &novel.Video{
		ID:        videoID,
		NovelID:   chapter.NovelID,
		UserID:    chapter.UserID,
		VideoType: novel.VideoTypeShot,
		ShotID:    shot.ID,
		ChapterID: chapter.ID,
		Version:   nextVersion,
		Status:    novel.VideoStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.videoRepo.Create(ctx, video); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	// 5. 异步生成视频
	go func() {
		ctx := context.Background()
		if err := s.generateVideoForShot(ctx, videoID, shot, chapter, nextVersion); err != nil {
			log.Error().
				Err(err).
				Str("shot_id", shotID).
				Str("video_id", videoID).
				Msg("生成视频失败")
			// 更新状态为失败
			if updateErr := s.videoRepo.UpdateStatus(ctx, videoID, novel.VideoStatusFailed, err.Error()); updateErr != nil {
				log.Error().
					Err(updateErr).
					Str("video_id", videoID).
					Msg("更新视频状态失败")
			}
		}
	}()

	return videoID, nil
}

// GetVideosByShot 获取镜头的所有视频
func (s *novelService) GetVideosByShot(ctx context.Context, shotID string) ([]*novel.Video, error) {
	return s.videoRepo.FindByShotID(ctx, shotID)
}

// getFinishVideoPath 获取 finish.mp4 文件路径
// 优先从环境变量 FINISH_VIDEO_PATH 获取，否则使用默认路径
func (s *novelService) getFinishVideoPath() string {
	// 从环境变量获取
	if path := os.Getenv("FINISH_VIDEO_PATH"); path != "" {
		return path
	}

	// 默认路径（相对于项目根目录）
	// 注意：这里假设 finish.mp4 在项目根目录的 src/banner/ 目录下
	// 实际使用时可能需要根据项目结构调整
	defaultPath := "src/banner/finish_compatible.mp4"

	// 检查文件是否存在
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	// 如果默认路径不存在，返回空字符串（表示跳过 finish 视频）
	return ""
}

// enhanceVideoPrompt 增强已有的 video_prompt
// 结合解说内容和场景描述，使视频 prompt 更加丰富和详细
func enhanceVideoPrompt(baseVideoPrompt, imagePrompt, scenePrompt, narration string) string {
	// 如果基础 prompt 为空，回退到完全构建的方式
	if baseVideoPrompt == "" {
		return buildVideoPromptFromImage(imagePrompt, scenePrompt, narration)
	}

	// 提取基础 prompt 中的关键信息（如时长、景别、镜头运动等）
	// 然后结合解说内容和场景描述进行增强
	var enhancedParts []string

	// 1. 保留基础 prompt 中的核心信息（时长、景别、镜头运动等）
	// 检查是否包含时长信息
	if strings.Contains(baseVideoPrompt, "时长") {
		// 提取时长信息（如"时长8秒"）
		enhancedParts = append(enhancedParts, baseVideoPrompt)
	} else {
		// 如果没有时长信息，添加基础 prompt
		enhancedParts = append(enhancedParts, baseVideoPrompt)
	}

	// 2. 从解说内容中提取动作和情绪描述
	actionKeywords := map[string]string{
		"走":  "人物缓慢行走，步伐自然",
		"跑":  "人物快速奔跑，动作幅度大",
		"跳":  "人物跳跃动作，充满动感",
		"转身": "人物缓缓转身，动作流畅",
		"回头": "人物缓缓回头，眼神自然",
		"抬头": "人物抬头动作，表情自然",
		"低头": "人物低头动作，神态专注",
		"观察": "人物仔细观察，眼神专注",
		"看":  "人物目光专注，表情自然",
		"望":  "人物远望，眼神深邃",
		"抬手": "人物抬手动作，手势自然",
		"挥手": "人物挥手示意，动作优雅",
		"点头": "人物点头示意，表情自然",
		"摇头": "人物摇头动作，表情生动",
		"移动": "人物位置移动，画面动态",
		"前进": "人物向前移动，步伐稳健",
		"后退": "人物向后移动，动作自然",
		"坐下": "人物坐下动作，姿态自然",
		"站起": "人物站起动作，动作流畅",
		"伸手": "人物伸手动作，手势自然",
		"握拳": "人物握拳动作，充满力量",
		"张开": "人物张开手臂，动作舒展",
	}

	hasAction := false
	for keyword, actionDesc := range actionKeywords {
		if strings.Contains(narration, keyword) {
			// 检查是否已经在基础 prompt 中包含类似描述
			if !strings.Contains(baseVideoPrompt, keyword) && !strings.Contains(baseVideoPrompt, actionDesc) {
				enhancedParts = append(enhancedParts, actionDesc)
				hasAction = true
				break
			}
		}
	}

	// 3. 从解说内容中提取情绪和表情描述
	emotionKeywords := map[string]string{
		"笑":  "人物表情自然，面带微笑",
		"哭":  "人物表情悲伤，情绪真实",
		"怒":  "人物表情严肃，情绪强烈",
		"惊":  "人物表情惊讶，反应自然",
		"疑惑": "人物表情疑惑，眼神专注",
		"思考": "人物表情沉思，神态自然",
		"温柔": "人物表情温柔，神态柔和",
		"坚定": "人物表情坚定，眼神有力",
		"兴奋": "人物表情兴奋，情绪高涨",
		"紧张": "人物表情紧张，神态不安",
		"放松": "人物表情放松，神态自然",
		"专注": "人物表情专注，眼神集中",
		"困惑": "人物表情困惑，神态迷茫",
		"期待": "人物表情期待，眼神明亮",
		"失望": "人物表情失望，情绪低落",
	}

	for keyword, emotionDesc := range emotionKeywords {
		if strings.Contains(narration, keyword) {
			// 检查是否已经在基础 prompt 中包含类似描述
			if !strings.Contains(baseVideoPrompt, keyword) && !strings.Contains(baseVideoPrompt, emotionDesc) {
				enhancedParts = append(enhancedParts, emotionDesc)
				break
			}
		}
	}

	// 4. 从场景描述中提取环境动态效果
	if strings.Contains(scenePrompt, "风") || strings.Contains(imagePrompt, "风") || strings.Contains(narration, "风") {
		if !strings.Contains(baseVideoPrompt, "风") {
			enhancedParts = append(enhancedParts, "背景有风吹动，树叶或衣物轻微摆动")
		}
	} else if strings.Contains(scenePrompt, "雨") || strings.Contains(imagePrompt, "雨") || strings.Contains(narration, "雨") {
		if !strings.Contains(baseVideoPrompt, "雨") {
			enhancedParts = append(enhancedParts, "背景有雨滴落下，画面湿润自然")
		}
	} else if strings.Contains(scenePrompt, "雪") || strings.Contains(imagePrompt, "雪") || strings.Contains(narration, "雪") {
		if !strings.Contains(baseVideoPrompt, "雪") {
			enhancedParts = append(enhancedParts, "背景有雪花飘落，画面唯美")
		}
	} else if !hasAction && !strings.Contains(baseVideoPrompt, "背景") {
		enhancedParts = append(enhancedParts, "背景有轻微的运动感，光影自然变化")
	}

	// 5. 从解说内容中提取节奏描述
	if strings.Contains(narration, "缓缓") || strings.Contains(narration, "慢慢") || strings.Contains(narration, "缓慢") {
		if !strings.Contains(baseVideoPrompt, "缓慢") && !strings.Contains(baseVideoPrompt, "缓缓") {
			enhancedParts = append(enhancedParts, "整体节奏缓慢，画面过渡自然流畅")
		}
	} else if strings.Contains(narration, "快速") || strings.Contains(narration, "迅速") || strings.Contains(narration, "急速") {
		if !strings.Contains(baseVideoPrompt, "快速") && !strings.Contains(baseVideoPrompt, "迅速") {
			enhancedParts = append(enhancedParts, "整体节奏较快，动作流畅有力")
		}
	}

	// 6. 添加画面质量描述（如果基础 prompt 中没有）
	if !strings.Contains(baseVideoPrompt, "清晰") && !strings.Contains(baseVideoPrompt, "细节") {
		enhancedParts = append(enhancedParts, "画面清晰，细节丰富，动态效果自然")
	}

	// 组合所有部分
	if len(enhancedParts) > 0 {
		return strings.Join(enhancedParts, "，")
	}

	// 如果没有增强内容，返回基础 prompt
	return baseVideoPrompt
}

// buildVideoPromptFromImage 基于图片 prompt 和场景描述构建视频动态效果 prompt
// 添加镜头运动、转场效果、动作描述等，使生成的视频有更丰富的动态效果
func buildVideoPromptFromImage(imagePrompt, scenePrompt, narration string) string {
	// 如果图片 prompt 为空，使用场景 prompt
	if imagePrompt == "" {
		imagePrompt = scenePrompt
	}

	// 如果都为空，返回空字符串（调用方会使用默认 prompt）
	if imagePrompt == "" {
		return ""
	}

	// 构建详细的视频动态效果描述
	// 基于图片内容、场景描述和解说内容，生成更详细的动态效果描述
	var promptParts []string

	// 1. 基础动态效果
	promptParts = append(promptParts, "画面有明显的动态效果")

	// 2. 镜头运动描述
	// 根据场景描述判断镜头类型
	if strings.Contains(scenePrompt, "近景") || strings.Contains(imagePrompt, "近景") || strings.Contains(scenePrompt, "特写") || strings.Contains(imagePrompt, "特写") {
		promptParts = append(promptParts, "镜头缓慢推进，聚焦人物细节")
	} else if strings.Contains(scenePrompt, "远景") || strings.Contains(imagePrompt, "远景") {
		promptParts = append(promptParts, "镜头缓慢拉远，展现全景")
	} else if strings.Contains(scenePrompt, "中景") || strings.Contains(imagePrompt, "中景") {
		promptParts = append(promptParts, "镜头平稳移动，保持中景构图")
	} else {
		promptParts = append(promptParts, "镜头缓慢推进，画面自然过渡")
	}

	// 3. 人物动作描述
	actionKeywords := map[string]string{
		"走":  "人物缓慢行走，步伐自然",
		"跑":  "人物快速奔跑，动作幅度大",
		"跳":  "人物跳跃动作，充满动感",
		"转身": "人物缓缓转身，动作流畅",
		"回头": "人物缓缓回头，眼神自然",
		"抬手": "人物抬手动作，手势自然",
		"挥手": "人物挥手示意，动作优雅",
		"点头": "人物点头示意，表情自然",
		"摇头": "人物摇头动作，表情生动",
		"移动": "人物位置移动，画面动态",
		"前进": "人物向前移动，步伐稳健",
		"后退": "人物向后移动，动作自然",
		"坐下": "人物坐下动作，姿态自然",
		"站起": "人物站起动作，动作流畅",
		"伸手": "人物伸手动作，手势自然",
		"握拳": "人物握拳动作，充满力量",
		"张开": "人物张开手臂，动作舒展",
	}

	hasAction := false
	for keyword, actionDesc := range actionKeywords {
		if strings.Contains(scenePrompt, keyword) || strings.Contains(imagePrompt, keyword) || strings.Contains(narration, keyword) {
			promptParts = append(promptParts, actionDesc)
			hasAction = true
			break
		}
	}

	// 4. 表情和情绪描述
	emotionKeywords := map[string]string{
		"笑":  "人物表情自然，面带微笑",
		"哭":  "人物表情悲伤，情绪真实",
		"怒":  "人物表情严肃，情绪强烈",
		"惊":  "人物表情惊讶，反应自然",
		"疑惑": "人物表情疑惑，眼神专注",
		"思考": "人物表情沉思，神态自然",
		"温柔": "人物表情温柔，神态柔和",
		"坚定": "人物表情坚定，眼神有力",
	}

	for keyword, emotionDesc := range emotionKeywords {
		if strings.Contains(narration, keyword) || strings.Contains(scenePrompt, keyword) {
			promptParts = append(promptParts, emotionDesc)
			break
		}
	}

	// 5. 背景和环境动态
	if strings.Contains(scenePrompt, "风") || strings.Contains(imagePrompt, "风") || strings.Contains(narration, "风") {
		promptParts = append(promptParts, "背景有风吹动，树叶或衣物轻微摆动")
	} else if strings.Contains(scenePrompt, "雨") || strings.Contains(imagePrompt, "雨") || strings.Contains(narration, "雨") {
		promptParts = append(promptParts, "背景有雨滴落下，画面湿润自然")
	} else if strings.Contains(scenePrompt, "雪") || strings.Contains(imagePrompt, "雪") || strings.Contains(narration, "雪") {
		promptParts = append(promptParts, "背景有雪花飘落，画面唯美")
	} else if !hasAction {
		promptParts = append(promptParts, "背景有轻微的运动感，光影自然变化")
	}

	// 6. 速度描述
	if strings.Contains(narration, "缓缓") || strings.Contains(narration, "慢慢") || strings.Contains(narration, "缓慢") {
		promptParts = append(promptParts, "整体节奏缓慢，画面过渡自然流畅")
	} else if strings.Contains(narration, "快速") || strings.Contains(narration, "迅速") || strings.Contains(narration, "急速") {
		promptParts = append(promptParts, "整体节奏较快，动作流畅有力")
	} else {
		promptParts = append(promptParts, "整体画面流畅自然，动作协调")
	}

	// 7. 画面质量
	promptParts = append(promptParts, "画面清晰，细节丰富，动态效果自然")

	// 组合所有部分
	videoPrompt := strings.Join(promptParts, "，")

	return videoPrompt
}

// abs 计算绝对值（用于时长差异计算）
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
