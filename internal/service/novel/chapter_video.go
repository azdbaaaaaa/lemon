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

	// GenerateFinalVideoForChapter 生成章节的最终完整视频（对应 concat_finish_video.py）
	// 拼接所有 narration 视频，添加 finish.mp4
	GenerateFinalVideoForChapter(ctx context.Context, chapterID string) (string, error)

	// GetVideoVersions 获取章节的所有视频版本号
	GetVideoVersions(ctx context.Context, chapterID string) ([]int, error)

	// GetVideosByStatus 根据状态查询视频（用于轮询）
	GetVideosByStatus(ctx context.Context, status string) ([]*novel.ChapterVideo, error)
}

// GenerateFirstVideosForChapter 已废弃：现在所有视频都使用图生视频方式，不再需要 first_video
// DEPRECATED: 使用 GenerateNarrationVideosForChapter 即可，所有视频都通过图生视频生成
func (s *novelService) GenerateFirstVideosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	return nil, fmt.Errorf("GenerateFirstVideosForChapter is deprecated, use GenerateNarrationVideosForChapter instead")
}

// GenerateNarrationVideosForChapter 为章节生成所有 narration 视频
// 对应 Python: concat_narration_video.py
// 逻辑：
//   - 从 ChapterNarration.Content.Scenes[].Shots[] 中提取所有 Shots
//   - 按照顺序为每个场景生成视频
//   - 内部实现决定：前3个场景合并成一个视频，其他场景每个单独生成视频
//   - 所有视频都使用图生视频方式（从图片生成视频）
func (s *novelService) GenerateNarrationVideosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	// 1. 获取章节的 narration
	narration, err := s.narrationRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find narration: %w", err)
	}

	if narration.Content == nil || len(narration.Content.Scenes) == 0 {
		return nil, fmt.Errorf("narration content is empty")
	}

	// 3. 从 Scenes[].Shots[] 中提取所有 Shots，按照顺序编号
	var allShots []struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.NarrationShot
		Index       int // narration 编号（从1开始）
	}

	narrationIndex := 1
	for _, scene := range narration.Content.Scenes {
		if scene == nil || len(scene.Shots) == 0 {
			continue
		}
		for _, shot := range scene.Shots {
			if shot == nil {
				continue
			}
			allShots = append(allShots, struct {
				SceneNumber string
				ShotNumber  string
				Shot        *novel.NarrationShot
				Index       int
			}{
				SceneNumber: scene.SceneNumber,
				ShotNumber:  shot.CloseupNumber,
				Shot:        shot,
				Index:       narrationIndex,
			})
			narrationIndex++
		}
	}

	if len(allShots) == 0 {
		return nil, fmt.Errorf("no shots found in narration content")
	}

	// 4. 自动生成下一个版本号
	videoVersion, err := s.getNextVideoVersion(ctx, chapterID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get next video version: %w", err)
	}

	// 5. 初始化 FFmpeg 客户端
	ffmpegClient := ffmpeg.NewClient()

	var videoIDs []string

	// 6. 为每个场景生成视频
	// 内部实现：前3个场景合并成一个视频，其他场景每个单独生成视频
	// 所有视频都使用图生视频方式
	if len(allShots) >= 3 {
		// 前3个场景合并成一个视频
		mergedVideoID, err := s.generateMergedNarrationVideo(ctx, chapterID, narration, allShots[0:3], nil, videoVersion, ffmpegClient)
		if err != nil {
			log.Error().Err(err).Msg("生成合并视频失败（前3个场景）")
		} else {
			videoIDs = append(videoIDs, mergedVideoID)
		}
	}

	// 其他场景每个单独生成视频
	if len(allShots) > 3 {
		for i := 3; i < len(allShots) && i < 30; i++ {
			shotInfo := allShots[i]
			narrationNum := fmt.Sprintf("%02d", shotInfo.Index)

			videoID, err := s.generateSingleNarrationVideo(ctx, chapterID, narration, shotInfo, narrationNum, videoVersion, ffmpegClient)
			if err != nil {
				log.Error().Err(err).Str("narration_num", narrationNum).Msg("生成场景视频失败")
				continue
			}
			videoIDs = append(videoIDs, videoID)
		}
	}

	return videoIDs, nil
}

// generateNarration01Video 已废弃：现在所有视频都使用图生视频方式，不再需要 first_video
// DEPRECATED: 此函数已不再使用，narration_01-03 现在通过 generateMergedNarrationVideo 统一生成
func (s *novelService) generateNarration01Video(
	ctx context.Context,
	narration *novel.ChapterNarration,
	video1 *novel.ChapterVideo,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	return "", fmt.Errorf("generateNarration01Video is deprecated, use generateMergedNarrationVideo instead")
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
// 对应 Python: create_merged_narration_video()
// 逻辑：
//   - 完全使用图生视频方式（每个场景的图片按对应音频时长生成视频）
//   - 合并多个场景的视频、字幕和音频
//   - 添加 BGM 和音效（可选）
func (s *novelService) generateMergedNarrationVideo(
	ctx context.Context,
	chapterID string,
	narration *novel.ChapterNarration,
	shots []struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.NarrationShot
		Index       int
	},
	video1 *novel.ChapterVideo, // 保留参数以保持接口兼容，但不再使用
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	if len(shots) != 3 {
		return "", fmt.Errorf("merged video requires exactly 3 shots, got %d", len(shots))
	}

	// 1. 获取前三个 Shots 的音频（sequence=1, 2, 3）
	audios, err := s.audioRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find audios: %w", err)
	}

	// 按 sequence 排序并获取前三个
	sort.Slice(audios, func(i, j int) bool {
		return audios[i].Sequence < audios[j].Sequence
	})

	if len(audios) < 3 {
		return "", fmt.Errorf("need at least 3 audio segments for merged narration, got %d", len(audios))
	}

	// 计算总音频时长
	var totalAudioDuration float64
	for i := 0; i < 3; i++ {
		audioDuration := audios[i].Duration
		if audioDuration <= 0 {
			// TODO: 修复音频 duration 为 0 的问题，确保 TTS API 返回的 duration 正确解析并保存到数据库
			// 当前临时方案：如果音频时长为 0，使用默认值 10 秒
			audioDuration = 10.0
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", audios[i].Sequence).
				Msg("音频 duration 为 0，使用默认值 10 秒")
		}
		totalAudioDuration += audioDuration
	}

	if totalAudioDuration <= 0 {
		// TODO: 修复音频 duration 为 0 的问题，确保 TTS API 返回的 duration 正确解析并保存到数据库
		// 当前临时方案：如果总时长为 0，使用默认值 30 秒（3个音频片段 × 10秒）
		totalAudioDuration = 30.0
		log.Warn().
			Str("narration_id", narration.ID).
			Msg("总音频 duration 为 0，使用默认值 30 秒")
	}

	// 2. 创建临时目录
	tmpDir := os.TempDir()

	// 3. 下载前三个音频片段对应的字幕文件并合并
	// 获取前三个音频片段的字幕
	var subtitlePaths []string
	for i := 0; i < 3; i++ {
		subtitle, err := s.subtitleRepo.FindByNarrationIDAndSequence(ctx, narration.ID, audios[i].Sequence)
		if err != nil {
			return "", fmt.Errorf("find subtitle for sequence %d: %w", audios[i].Sequence, err)
		}

		// 下载字幕文件
		subtitleDownloadReq := &service.DownloadFileRequest{
			ResourceID: subtitle.SubtitleResourceID,
			UserID:     narration.UserID,
		}
		subtitleResult, err := s.resourceService.DownloadFile(ctx, subtitleDownloadReq)
		if err != nil {
			return "", fmt.Errorf("download subtitle %d: %w", i+1, err)
		}
		defer subtitleResult.Data.Close()

		// 保存字幕到临时文件
		tmpSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("subtitle_%d_%s.ass", i+1, id.New()))
		defer os.Remove(tmpSubtitlePath)
		subtitleFile, err := os.Create(tmpSubtitlePath)
		if err != nil {
			return "", fmt.Errorf("create temp subtitle file %d: %w", i+1, err)
		}
		if _, err := io.Copy(subtitleFile, subtitleResult.Data); err != nil {
			subtitleFile.Close()
			return "", fmt.Errorf("copy subtitle data %d: %w", i+1, err)
		}
		subtitleFile.Close()
		subtitlePaths = append(subtitlePaths, tmpSubtitlePath)
	}

	// 合并三个字幕文件（合并 ASS 文件的 Dialogue 事件）
	tmpMergedSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("subtitle_merged_%s.ass", id.New()))
	defer os.Remove(tmpMergedSubtitlePath)
	if err := s.mergeASSFiles(ctx, subtitlePaths, tmpMergedSubtitlePath); err != nil {
		return "", fmt.Errorf("merge ASS files: %w", err)
	}

	// 4. 创建视频片段列表（完全使用图生视频）
	var videoSegmentPaths []string
	defer func() {
		for _, path := range videoSegmentPaths {
			os.Remove(path)
		}
	}()

	// 3.1 使用 image_01-03 按音频时长分配（每个图片对应一个音频片段）
	// 收集所有图片的 prompt（用于视频记录）
	var imagePrompts []string
	var images []*novel.ChapterImage
	for i, shotInfo := range shots {
		// 根据 scene_number 和 shot_number 查找图片
		image, err := s.chapterImageRepo.FindBySceneAndShot(ctx, chapterID, shotInfo.SceneNumber, shotInfo.ShotNumber)
		if err != nil {
			return "", fmt.Errorf("find image for shot %d (scene=%s, shot=%s): %w", i+1, shotInfo.SceneNumber, shotInfo.ShotNumber, err)
		}

		// 保存图片引用和 prompt
		images = append(images, image)
		if image.Prompt != "" {
			imagePrompts = append(imagePrompts, image.Prompt)
		}

		// 下载图片
		imageDownloadReq := &service.DownloadFileRequest{
			ResourceID: image.ImageResourceID,
			UserID:     narration.UserID,
		}
		imageResult, err := s.resourceService.DownloadFile(ctx, imageDownloadReq)
		if err != nil {
			return "", fmt.Errorf("download image %d: %w", i+1, err)
		}
		defer imageResult.Data.Close()

		tmpImagePath := filepath.Join(tmpDir, fmt.Sprintf("image_%d_%s.jpg", i+1, id.New()))
		imageFile, err := os.Create(tmpImagePath)
		if err != nil {
			return "", fmt.Errorf("create temp image file: %w", err)
		}
		if _, err := io.Copy(imageFile, imageResult.Data); err != nil {
			imageFile.Close()
			return "", fmt.Errorf("copy image data: %w", err)
		}
		imageFile.Close()

		// 从图片创建视频（使用对应音频片段的时长）
		// 注意：这里使用每个音频片段的实际时长，而不是平均分配
		audioDuration := audios[i].Duration
		if audioDuration <= 0 {
			// TODO: 修复音频 duration 为 0 的问题，确保 TTS API 返回的 duration 正确解析并保存到数据库
			// 当前临时方案：如果音频时长为 0，使用默认值 10 秒
			audioDuration = 10.0
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", audios[i].Sequence).
				Int("shot_index", i+1).
				Msg("音频 duration 为 0，使用默认值 10 秒")
		}

		tmpImageVideoPath := filepath.Join(tmpDir, fmt.Sprintf("image_video_%d_%s.mp4", i+1, id.New()))
		if err := ffmpegClient.CreateImageVideo(ctx, tmpImagePath, tmpImageVideoPath, audioDuration, 720, 1280, 30); err != nil {
			return "", fmt.Errorf("create image video %d: %w", i+1, err)
		}
		videoSegmentPaths = append(videoSegmentPaths, tmpImageVideoPath)
		os.Remove(tmpImagePath) // 清理图片文件
	}

	if len(videoSegmentPaths) == 0 {
		return "", fmt.Errorf("no video segments generated")
	}

	// 4. 合并视频片段
	tmpMergedVideoPath := filepath.Join(tmpDir, fmt.Sprintf("merged_video_%s.mp4", id.New()))
	defer os.Remove(tmpMergedVideoPath)

	if err := ffmpegClient.ConcatVideos(ctx, videoSegmentPaths, tmpMergedVideoPath); err != nil {
		return "", fmt.Errorf("concat video segments: %w", err)
	}

	// 5. 合并前三个音频文件
	tmpMergedAudioPath := filepath.Join(tmpDir, fmt.Sprintf("merged_audio_%s.mp3", id.New()))
	defer os.Remove(tmpMergedAudioPath)

	// 下载三个音频文件
	var audioPaths []string
	for i := 0; i < 3; i++ {
		audioDownloadReq := &service.DownloadFileRequest{
			ResourceID: audios[i].AudioResourceID,
			UserID:     narration.UserID,
		}
		audioResult, err := s.resourceService.DownloadFile(ctx, audioDownloadReq)
		if err != nil {
			return "", fmt.Errorf("download audio %d: %w", i+1, err)
		}
		defer audioResult.Data.Close()

		tmpAudioPath := filepath.Join(tmpDir, fmt.Sprintf("audio_%d_%s.mp3", i+1, id.New()))
		audioFile, err := os.Create(tmpAudioPath)
		if err != nil {
			return "", fmt.Errorf("create temp audio file: %w", err)
		}
		if _, err := io.Copy(audioFile, audioResult.Data); err != nil {
			audioFile.Close()
			return "", fmt.Errorf("copy audio data: %w", err)
		}
		audioFile.Close()
		audioPaths = append(audioPaths, tmpAudioPath)
		defer os.Remove(tmpAudioPath)
	}

	// 使用 FFmpeg 合并音频
	if err := s.mergeAudioFiles(ctx, audioPaths, tmpMergedAudioPath); err != nil {
		return "", fmt.Errorf("merge audio files: %w", err)
	}

	// 7. 添加字幕到视频
	tmpWithSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("video_subtitle_%s.mp4", id.New()))
	defer os.Remove(tmpWithSubtitlePath)

	if err := ffmpegClient.AddSubtitles(ctx, tmpMergedVideoPath, tmpMergedSubtitlePath, tmpWithSubtitlePath); err != nil {
		return "", fmt.Errorf("add subtitles: %w", err)
	}

	// 8. 替换音频
	tmpFinalPath := filepath.Join(tmpDir, fmt.Sprintf("video_final_%s.mp4", id.New()))
	defer os.Remove(tmpFinalPath)

	if err := s.replaceVideoAudio(ctx, tmpWithSubtitlePath, tmpMergedAudioPath, tmpFinalPath, ffmpegClient); err != nil {
		return "", fmt.Errorf("replace audio: %w", err)
	}

	// 9. 标准化视频分辨率
	tmpStandardizedPath := filepath.Join(tmpDir, fmt.Sprintf("video_std_%s.mp4", id.New()))
	defer os.Remove(tmpStandardizedPath)

	if err := ffmpegClient.StandardizeVideo(ctx, tmpFinalPath, tmpStandardizedPath, 720, 1280, 30); err != nil {
		return "", fmt.Errorf("standardize video: %w", err)
	}

	// 10. 上传最终视频到 resource 模块
	finalVideoFile, err := os.Open(tmpStandardizedPath)
	if err != nil {
		return "", fmt.Errorf("open final video: %w", err)
	}
	defer finalVideoFile.Close()

	fileName := fmt.Sprintf("%s_narration_01-03_video.mp4", chapterID)
	uploadReq := &service.UploadFileRequest{
		UserID:      narration.UserID,
		FileName:    fileName,
		ContentType: "video/mp4",
		Ext:         "mp4",
		Data:        finalVideoFile,
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload video: %w", err)
	}

	// 11. 创建视频记录
	videoID := id.New()

	// 构建 prompt：合并所有图片的 prompt
	var videoPrompt string
	if len(imagePrompts) > 0 {
		videoPrompt = strings.Join(imagePrompts, "; ")
	} else {
		videoPrompt = "图生视频（前3个场景合并）"
	}

	videoEntity := &novel.ChapterVideo{
		ID:              videoID,
		ChapterID:       chapterID,
		NarrationID:     narration.ID,
		UserID:          narration.UserID,
		Sequence:        1, // 合并视频的 sequence 为 1
		VideoResourceID: uploadResult.ResourceID,
		Duration:        totalAudioDuration,
		VideoType:       "narration_video",
		Prompt:          videoPrompt,
		Version:         version,
		Status:          "completed",
	}

	if err := s.videoRepo.Create(ctx, videoEntity); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	return videoID, nil
}

// generateSingleNarrationVideo 生成单个场景的视频（内部实现：第4个场景及之后每个单独生成）
func (s *novelService) generateSingleNarrationVideo(
	ctx context.Context,
	chapterID string,
	narration *novel.ChapterNarration,
	shotInfo struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.NarrationShot
		Index       int
	},
	narrationNum string,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	// 1. 根据 scene_number 和 shot_number 查找图片
	image, err := s.chapterImageRepo.FindBySceneAndShot(ctx, chapterID, shotInfo.SceneNumber, shotInfo.ShotNumber)
	if err != nil {
		return "", fmt.Errorf("find image: %w", err)
	}

	// 2. 获取对应的音频（通过 sequence 匹配）
	audios, err := s.audioRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find audios: %w", err)
	}

	// 找到对应 sequence 的音频（narration_04 对应 sequence=4）
	var audio *novel.ChapterAudio
	for _, a := range audios {
		if a.Sequence == shotInfo.Index {
			audio = a
			break
		}
	}

	if audio == nil {
		return "", fmt.Errorf("audio not found for sequence %d", shotInfo.Index)
	}

	audioDuration := audio.Duration
	if audioDuration <= 0 {
		// TODO: 修复音频 duration 为 0 的问题，确保 TTS API 返回的 duration 正确解析并保存到数据库
		// 当前临时方案：如果音频时长为 0，使用默认值 10 秒
		audioDuration = 10.0
		log.Warn().
			Str("narration_id", narration.ID).
			Int("sequence", audio.Sequence).
			Str("narration_num", narrationNum).
			Msg("音频 duration 为 0，使用默认值 10 秒")
	}

	// 3. 下载图片
	imageDownloadReq := &service.DownloadFileRequest{
		ResourceID: image.ImageResourceID,
		UserID:     narration.UserID,
	}
	imageResult, err := s.resourceService.DownloadFile(ctx, imageDownloadReq)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer imageResult.Data.Close()

	tmpDir := os.TempDir()
	tmpImagePath := filepath.Join(tmpDir, fmt.Sprintf("image_%s.jpg", id.New()))
	defer os.Remove(tmpImagePath)
	imageFile, err := os.Create(tmpImagePath)
	if err != nil {
		return "", fmt.Errorf("create temp image file: %w", err)
	}
	if _, err := io.Copy(imageFile, imageResult.Data); err != nil {
		imageFile.Close()
		return "", fmt.Errorf("copy image data: %w", err)
	}
	imageFile.Close()

	// 读取图片数据，转换为 base64 data URL
	imageData, err := os.ReadFile(tmpImagePath)
	if err != nil {
		return "", fmt.Errorf("read image file: %w", err)
	}
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)
	imageDataURL := fmt.Sprintf("data:image/jpeg;base64,%s", imageBase64)

	// 4. 构建视频 prompt
	// 优先使用 shot.VideoPrompt（如果 LLM 生成了专门的视频 prompt）
	// 如果没有，则基于图片 prompt 和场景描述构建
	var videoPrompt string
	if shotInfo.Shot.VideoPrompt != "" {
		videoPrompt = shotInfo.Shot.VideoPrompt
		log.Info().
			Str("scene", shotInfo.SceneNumber).
			Str("shot", shotInfo.ShotNumber).
			Str("video_prompt", videoPrompt).
			Msg("使用 LLM 生成的 video_prompt")
	} else {
		// 如果没有 video_prompt，基于图片 prompt 和场景描述构建
		videoPrompt = buildVideoPromptFromImage(image.Prompt, shotInfo.Shot.ScenePrompt, shotInfo.Shot.Narration)
		if videoPrompt == "" {
			videoPrompt = "画面有明显的动态效果，镜头缓慢推进，人物有自然的动作和表情变化，背景有轻微的运动感，整体画面流畅自然"
		}
		log.Info().
			Str("scene", shotInfo.SceneNumber).
			Str("shot", shotInfo.ShotNumber).
			Str("video_prompt", videoPrompt).
			Msg("使用构建的 video_prompt")
	}

	// 5. 从图片创建视频
	// 如果音频时长 <= 12 秒，使用 Ark API 生成视频（使用 videoPrompt）
	// 如果音频时长 > 12 秒，使用 FFmpeg 从图片创建视频（Ken Burns 效果）
	tmpVideoPath := filepath.Join(tmpDir, fmt.Sprintf("video_%s.mp4", id.New()))
	defer os.Remove(tmpVideoPath)

	if audioDuration <= 12.0 {
		// 使用 Ark API 生成视频（限制最大 12 秒）
		limitedDuration := int(audioDuration)
		videoData, err := s.videoProvider.GenerateVideoFromImage(ctx, imageDataURL, limitedDuration, videoPrompt)
		if err != nil {
			return "", fmt.Errorf("generate video from image: %w", err)
		}

		// 保存视频数据到临时文件
		if err := os.WriteFile(tmpVideoPath, videoData, 0644); err != nil {
			return "", fmt.Errorf("save video file: %w", err)
		}
	} else {
		// 音频时长超过 12 秒，使用 FFmpeg 从图片创建视频（Ken Burns 效果）
		// 参考 Python: create_image_video_with_effects
		log.Info().
			Float64("audio_duration", audioDuration).
			Msg("音频时长超过 12 秒，使用 FFmpeg 从图片创建视频")
		if err := ffmpegClient.CreateImageVideo(ctx, tmpImagePath, tmpVideoPath, audioDuration, 720, 1280, 30); err != nil {
			return "", fmt.Errorf("create image video: %w", err)
		}
	}

	// 6. 获取对应音频片段的字幕文件
	subtitle, err := s.subtitleRepo.FindByNarrationIDAndSequence(ctx, narration.ID, audio.Sequence)
	if err != nil {
		return "", fmt.Errorf("find subtitle for sequence %d: %w", audio.Sequence, err)
	}

	// 下载字幕文件
	subtitleDownloadReq := &service.DownloadFileRequest{
		ResourceID: subtitle.SubtitleResourceID,
		UserID:     narration.UserID,
	}
	subtitleResult, err := s.resourceService.DownloadFile(ctx, subtitleDownloadReq)
	if err != nil {
		return "", fmt.Errorf("download subtitle: %w", err)
	}
	defer subtitleResult.Data.Close()

	// 保存字幕到临时文件
	tmpSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("subtitle_%s.ass", id.New()))
	defer os.Remove(tmpSubtitlePath)
	subtitleFile, err := os.Create(tmpSubtitlePath)
	if err != nil {
		return "", fmt.Errorf("create temp subtitle file: %w", err)
	}
	if _, err := io.Copy(subtitleFile, subtitleResult.Data); err != nil {
		subtitleFile.Close()
		return "", fmt.Errorf("copy subtitle data: %w", err)
	}
	subtitleFile.Close()

	// 7. 添加字幕到视频
	tmpWithSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("video_subtitle_%s.mp4", id.New()))
	defer os.Remove(tmpWithSubtitlePath)

	if err := ffmpegClient.AddSubtitles(ctx, tmpVideoPath, tmpSubtitlePath, tmpWithSubtitlePath); err != nil {
		return "", fmt.Errorf("add subtitles: %w", err)
	}

	// 8. 下载音频文件
	audioDownloadReq := &service.DownloadFileRequest{
		ResourceID: audio.AudioResourceID,
		UserID:     narration.UserID,
	}
	audioResult, err := s.resourceService.DownloadFile(ctx, audioDownloadReq)
	if err != nil {
		return "", fmt.Errorf("download audio: %w", err)
	}
	defer audioResult.Data.Close()

	tmpAudioPath := filepath.Join(tmpDir, fmt.Sprintf("audio_%s.mp3", id.New()))
	defer os.Remove(tmpAudioPath)
	audioFile, err := os.Create(tmpAudioPath)
	if err != nil {
		return "", fmt.Errorf("create temp audio file: %w", err)
	}
	if _, err := io.Copy(audioFile, audioResult.Data); err != nil {
		audioFile.Close()
		return "", fmt.Errorf("copy audio data: %w", err)
	}
	audioFile.Close()

	// 8. 替换音频
	tmpFinalPath := filepath.Join(tmpDir, fmt.Sprintf("video_final_%s.mp4", id.New()))
	defer os.Remove(tmpFinalPath)

	if err := s.replaceVideoAudio(ctx, tmpWithSubtitlePath, tmpAudioPath, tmpFinalPath, ffmpegClient); err != nil {
		return "", fmt.Errorf("replace audio: %w", err)
	}

	// 9. 标准化视频分辨率
	tmpStandardizedPath := filepath.Join(tmpDir, fmt.Sprintf("video_std_%s.mp4", id.New()))
	defer os.Remove(tmpStandardizedPath)

	if err := ffmpegClient.StandardizeVideo(ctx, tmpFinalPath, tmpStandardizedPath, 720, 1280, 30); err != nil {
		return "", fmt.Errorf("standardize video: %w", err)
	}

	// 11. 上传视频
	finalVideoFile, err := os.Open(tmpStandardizedPath)
	if err != nil {
		return "", fmt.Errorf("open final video: %w", err)
	}
	defer finalVideoFile.Close()

	fileName := fmt.Sprintf("%s_narration_%s_video.mp4", chapterID, narrationNum)
	uploadReq := &service.UploadFileRequest{
		UserID:      narration.UserID,
		FileName:    fileName,
		ContentType: "video/mp4",
		Ext:         "mp4",
		Data:        finalVideoFile,
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload video: %w", err)
	}

	// 12. 创建视频记录
	videoID := id.New()
	// 获取当前已生成的视频数量（用于 sequence）
	existingVideos, _ := s.videoRepo.FindByChapterIDAndType(ctx, chapterID, "narration_video")
	sequence := len(existingVideos) + 1

	// videoPrompt 已经在前面（第 571 行）构建好了，这里直接使用

	videoEntity := &novel.ChapterVideo{
		ID:              videoID,
		ChapterID:       chapterID,
		NarrationID:     narration.ID,
		UserID:          narration.UserID,
		Sequence:        sequence,
		VideoResourceID: uploadResult.ResourceID,
		Duration:        audioDuration,
		VideoType:       "narration_video",
		Prompt:          videoPrompt,
		Version:         version,
		Status:          "completed",
	}

	if err := s.videoRepo.Create(ctx, videoEntity); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	return videoID, nil
}

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
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	// 2. 获取章节的所有 narration 视频
	narrationVideos, err := s.videoRepo.FindByChapterIDAndType(ctx, chapterID, "narration_video")
	if err != nil {
		return "", fmt.Errorf("find narration videos: %w", err)
	}

	if len(narrationVideos) == 0 {
		return "", fmt.Errorf("no narration videos found for chapter %s", chapterID)
	}

	// 按 sequence 排序
	sort.Slice(narrationVideos, func(i, j int) bool {
		return narrationVideos[i].Sequence < narrationVideos[j].Sequence
	})

	// 3. 初始化 FFmpeg 客户端
	ffmpegClient := ffmpeg.NewClient()

	// 4. 下载所有视频到临时文件
	tmpDir := os.TempDir()
	var videoPaths []string
	for i, video := range narrationVideos {
		downloadReq := &service.DownloadFileRequest{
			ResourceID: video.VideoResourceID,
			UserID:     chapter.UserID,
		}
		videoResult, err := s.resourceService.DownloadFile(ctx, downloadReq)
		if err != nil {
			return "", fmt.Errorf("download video %d: %w", i+1, err)
		}
		defer videoResult.Data.Close()

		tmpVideoPath := filepath.Join(tmpDir, fmt.Sprintf("video_%d_%s.mp4", i+1, id.New()))
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
	videoID := id.New()
	videoVersion, err := s.getNextVideoVersion(ctx, chapterID, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get next video version: %w", err)
	}

	videoEntity := &novel.ChapterVideo{
		ID:              videoID,
		ChapterID:       chapterID,
		UserID:          chapter.UserID,
		Sequence:        1,
		VideoResourceID: uploadResult.ResourceID,
		Duration:        totalDuration,
		VideoType:       "final_video",
		Version:         videoVersion,
		Status:          "completed",
	}

	if err := s.videoRepo.Create(ctx, videoEntity); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	return videoID, nil
}

// GetVideoVersions 获取章节的所有视频版本号
func (s *novelService) GetVideoVersions(ctx context.Context, chapterID string) ([]int, error) {
	return s.videoRepo.FindVersionsByChapterID(ctx, chapterID)
}

// GetVideosByStatus 根据状态查询视频（用于轮询）
func (s *novelService) GetVideosByStatus(ctx context.Context, status string) ([]*novel.ChapterVideo, error) {
	return s.videoRepo.FindByStatus(ctx, status)
}

// getNextVideoVersion 获取章节的下一个视频版本号（自动递增）
func (s *novelService) getNextVideoVersion(ctx context.Context, chapterID string, baseVersion int) (int, error) {
	versions, err := s.videoRepo.FindVersionsByChapterID(ctx, chapterID)
	if err != nil {
		// 如果没有找到任何版本，返回 1 或基础版本号
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	if len(versions) == 0 {
		if baseVersion == 0 {
			return 1, nil
		}
		return baseVersion, nil
	}

	// 如果指定了基础版本号，检查该版本是否已存在
	if baseVersion > 0 {
		for _, v := range versions {
			if v == baseVersion {
				// 该版本已存在，返回下一个版本号
				maxVersion := 0
				for _, v := range versions {
					if v > maxVersion {
						maxVersion = v
					}
				}
				return maxVersion + 1, nil
			}
		}
		// 该版本不存在，直接返回
		return baseVersion, nil
	}

	// 如果没有指定基础版本号，查找所有版本号中的最大值
	maxVersion := 0
	for _, v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}

	return maxVersion + 1, nil
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

	// 构建视频动态效果描述
	// 基于图片内容和场景描述，添加镜头运动、转场效果、动作描述
	videoPrompt := "画面有明显的动态效果，镜头缓慢推进，人物有自然的动作和表情变化"

	// 如果场景描述中包含动作关键词，增强动作描述
	actionKeywords := []string{"走", "跑", "跳", "转身", "回头", "抬手", "挥手", "点头", "摇头", "转身", "移动", "前进", "后退"}
	hasAction := false
	for _, keyword := range actionKeywords {
		if strings.Contains(scenePrompt, keyword) || strings.Contains(imagePrompt, keyword) {
			hasAction = true
			break
		}
	}

	if hasAction {
		videoPrompt += "，动作幅度较大，画面流畅自然"
	} else {
		videoPrompt += "，背景有轻微的运动感，整体画面流畅自然"
	}

	// 如果解说内容中包含情绪或动作描述，增强动态效果
	if strings.Contains(narration, "缓缓") || strings.Contains(narration, "慢慢") {
		videoPrompt += "，镜头缓慢推进，画面过渡自然"
	} else if strings.Contains(narration, "快速") || strings.Contains(narration, "迅速") {
		videoPrompt += "，画面变化较快，动作流畅"
	}

	return videoPrompt
}
