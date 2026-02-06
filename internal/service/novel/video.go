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
	"sync"

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

	// GenerateFinalVideoForChapterWithVersion 指定 narration 视频版本号，手动确认后再合并生成最终视频
	GenerateFinalVideoForChapterWithVersion(ctx context.Context, chapterID string, version int) (string, error)

	// GetVideoVersions 获取章节的所有视频版本号
	GetVideoVersions(ctx context.Context, chapterID string) ([]int, error)

	// GetVideosByStatus 根据状态查询视频（用于轮询）
	GetVideosByStatus(ctx context.Context, status novel.VideoStatus) ([]*novel.Video, error)

	// ListVideosByChapter 获取章节视频列表（可指定版本；version<=0 则取最新版本）
	ListVideosByChapter(ctx context.Context, chapterID string, version int) ([]*novel.Video, int, error)
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

	// 2. 从独立的表中查询场景和镜头
	scenes, err := s.sceneRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return nil, fmt.Errorf("find scenes: %w", err)
	}

	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes found for narration")
	}

	// 3. 从 Scenes 和 Shots 中提取所有 Shots，按照顺序编号
	var allShots []struct {
		SceneNumber string
		ShotNumber  string
		Shot        *novel.Shot
		Index       int // narration 编号（从1开始）
	}

	narrationIndex := 1
	for _, scene := range scenes {
		// 查询该场景下的所有镜头
		shots, err := s.shotRepo.FindBySceneID(ctx, scene.ID)
		if err != nil {
			continue
		}

		for _, shot := range shots {
			allShots = append(allShots, struct {
				SceneNumber string
				ShotNumber  string
				Shot        *novel.Shot
				Index       int
			}{
				SceneNumber: scene.SceneNumber,
				ShotNumber:  shot.ShotNumber,
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

	// 6. 并发为每个分镜生成视频（最大并发数：10）
	// 所有分镜都单独生成视频，使用图生视频方式
	maxConcurrency := 10
	maxShots := len(allShots)
	if maxShots > 30 {
		maxShots = 30
	}

	// 使用 channel 控制并发数
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var videoIDs []string
	var errors []error

	for i := 0; i < maxShots; i++ {
		shotInfo := allShots[i]
		narrationNum := fmt.Sprintf("%02d", shotInfo.Index)

		wg.Add(1)
		go func(shotInfo struct {
			SceneNumber string
			ShotNumber  string
			Shot        *novel.Shot
			Index       int
		}, narrationNum string) {
			defer wg.Done()

			// 获取信号量（限制并发数）
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			videoID, err := s.generateSingleNarrationVideo(ctx, chapterID, narration, shotInfo, narrationNum, videoVersion, ffmpegClient)
			if err != nil {
				log.Error().Err(err).Str("narration_num", narrationNum).Msg("生成分镜视频失败")
				mu.Lock()
				errors = append(errors, fmt.Errorf("narration %s: %w", narrationNum, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			videoIDs = append(videoIDs, videoID)
			mu.Unlock()
		}(shotInfo, narrationNum)
	}

	// 等待所有任务完成
	wg.Wait()

	// 如果有错误，记录日志但不返回错误（允许部分成功）
	if len(errors) > 0 {
		log.Warn().
			Int("total_shots", maxShots).
			Int("failed_count", len(errors)).
			Msg("部分分镜视频生成失败")
	}

	// 按 sequence 排序 videoIDs（确保顺序正确）
	// 由于每个 videoID 对应一个 shotInfo.Index，我们需要根据 video 的 sequence 排序
	// 但这里 videoIDs 的顺序已经和 shotInfo.Index 的顺序一致，所以不需要额外排序
	// 如果需要确保顺序，可以在生成后查询数据库按 sequence 排序

	return videoIDs, nil
}

// generateNarration01Video 已废弃：现在所有视频都使用图生视频方式，不再需要 first_video
// DEPRECATED: 此函数已不再使用，narration_01-03 现在通过 generateMergedNarrationVideo 统一生成
func (s *novelService) generateNarration01Video(
	ctx context.Context,
	narration *novel.Narration,
	video1 *novel.Video,
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
	narration *novel.Narration,
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
	var images []*novel.Image
	for i, shotInfo := range shots {
		// 根据 scene_number 和 shot_number 查找图片
		image, err := s.imageRepo.FindBySceneAndShot(ctx, chapterID, shotInfo.SceneNumber, shotInfo.ShotNumber)
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

	// 获取章节信息以获取 novel_id
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	videoEntity := &novel.Video{
		ID:          videoID,
		ChapterID:  chapterID,
		NarrationID: narration.ID,
		NovelID:    chapter.NovelID,
		UserID:     narration.UserID,
		Sequence:   1, // 合并视频的 sequence 为 1
		VideoResourceID: uploadResult.ResourceID,
		Duration:        totalAudioDuration,
		VideoType:       novel.VideoTypeNarration,
		Prompt:          videoPrompt,
		Version:         version,
		Status:          novel.VideoStatusCompleted,
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
	narration *novel.Narration,
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
	// 1. 优先使用分镜头的图片（Image 表）
	image, err := s.imageRepo.FindBySceneAndShot(ctx, chapterID, shotInfo.SceneNumber, shotInfo.ShotNumber)
	if err != nil {
		// 如果分镜头图片不存在，尝试使用角色图片或场景图片（简化逻辑：先不实现，直接返回错误）
		return "", fmt.Errorf("find image: %w", err)
	}

	// 2. 获取对应的音频（通过 sequence 匹配）
	audios, err := s.audioRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find audios: %w", err)
	}

	// 找到对应 sequence 的音频（narration_04 对应 sequence=4）
	var audio *novel.Audio
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

	// 4. 构建视频 prompt（简化逻辑：直接使用 shot 的 video_prompt，如果没有则使用默认值）
	videoPrompt := shotInfo.Shot.VideoPrompt
	if videoPrompt == "" {
		videoPrompt = "画面有明显的动态效果，镜头缓慢推进，人物有自然的动作和表情变化，背景有轻微的运动感，整体画面流畅自然"
	}

	// 5. 从图片创建视频
	// 参考 Python 版本：直接使用音频时长作为视频时长，不解析 video_prompt 中的时长
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

	// 6. 下载音频文件
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

	// 7. 获取对应音频片段的字幕文件
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

	// 7.5. 诊断：检查字幕时间戳和音频时长的同步情况
	// 用于排查为什么会出现字幕和音频不同步的问题
	subtitleContent, err := os.ReadFile(tmpSubtitlePath)
	if err != nil {
		log.Warn().Err(err).Msg("无法读取字幕文件，跳过字幕诊断")
	} else {
		// 解析 ASS 文件，提取第一个和最后一个字幕的时间戳
		subtitleLines := strings.Split(string(subtitleContent), "\n")
		var firstSubtitleTime, lastSubtitleTime float64
		var subtitleCount int

		for _, line := range subtitleLines {
			if strings.HasPrefix(line, "Dialogue:") {
				subtitleCount++
				// 解析 Dialogue 行：Dialogue: 0,Start,End,Style,Name,...
				parts := strings.Split(line, ",")
				if len(parts) >= 3 {
					// 解析开始时间
					if startTime, err := parseASSTime(parts[1]); err == nil {
						if firstSubtitleTime == 0 {
							firstSubtitleTime = startTime
						}
						lastSubtitleTime = startTime
					}
					// 解析结束时间
					if endTime, err := parseASSTime(parts[2]); err == nil {
						lastSubtitleTime = endTime
					}
				}
			}
		}

		log.Info().
			Str("narration_id", narration.ID).
			Int("sequence", audio.Sequence).
			Float64("audio_duration", audioDuration).
			Float64("subtitle_first_time", firstSubtitleTime).
			Float64("subtitle_last_time", lastSubtitleTime).
			Float64("subtitle_duration", lastSubtitleTime-firstSubtitleTime).
			Int("subtitle_count", subtitleCount).
			Msg("字幕同步诊断：对比音频时长和字幕时间戳范围")

		// 检查字幕时间戳是否覆盖整个音频时长
		if firstSubtitleTime > 0.5 {
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", audio.Sequence).
				Float64("first_subtitle_time", firstSubtitleTime).
				Msg("⚠️ 字幕开始时间不是从0开始，可能导致字幕延迟")
		}

		if lastSubtitleTime < audioDuration-0.5 {
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", audio.Sequence).
				Float64("audio_duration", audioDuration).
				Float64("last_subtitle_time", lastSubtitleTime).
				Float64("missing_duration", audioDuration-lastSubtitleTime).
				Msg("⚠️ 字幕结束时间早于音频结束时间，可能导致后半部分没有字幕")
		}
	}

	// 7.6. 诊断：检查视频实际时长和音频时长的差异
	videoInfo, err := ffmpegClient.GetVideoInfo(ctx, tmpVideoPath)
	if err != nil {
		log.Warn().Err(err).Msg("无法获取视频信息，跳过视频时长诊断")
	} else {
		actualVideoDuration := videoInfo.Duration
		durationDiff := actualVideoDuration - audioDuration
		log.Info().
			Str("narration_id", narration.ID).
			Int("sequence", audio.Sequence).
			Float64("audio_duration", audioDuration).
			Float64("video_duration", actualVideoDuration).
			Float64("duration_diff", durationDiff).
			Str("video_generation_method", func() string {
				if audioDuration <= 12.0 {
					return "Ark API"
				}
				return "FFmpeg (Ken Burns)"
			}()).
			Msg("视频时长诊断：对比音频和视频实际时长")

		if abs(durationDiff) > 0.5 {
			log.Warn().
				Str("narration_id", narration.ID).
				Int("sequence", audio.Sequence).
				Float64("audio_duration", audioDuration).
				Float64("video_duration", actualVideoDuration).
				Float64("duration_diff", durationDiff).
				Msg("⚠️ 视频时长和音频时长差异较大，可能导致字幕不匹配")
		}
	}

	// 8. 添加字幕到视频
	tmpWithSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("video_subtitle_%s.mp4", id.New()))
	defer os.Remove(tmpWithSubtitlePath)

	if err := ffmpegClient.AddSubtitles(ctx, tmpVideoPath, tmpSubtitlePath, tmpWithSubtitlePath); err != nil {
		return "", fmt.Errorf("add subtitles: %w", err)
	}

	// 9. 替换音频（参考 Python 版本：直接使用音频文件，FFmpeg 会自动处理时长对齐）
	tmpFinalPath := filepath.Join(tmpDir, fmt.Sprintf("video_final_%s.mp4", id.New()))
	defer os.Remove(tmpFinalPath)

	if err := s.replaceVideoAudio(ctx, tmpWithSubtitlePath, tmpAudioPath, tmpFinalPath, ffmpegClient); err != nil {
		return "", fmt.Errorf("replace audio: %w", err)
	}

	// 12. 标准化视频分辨率
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
	// 使用 shotInfo.Index 作为 sequence，确保与分镜顺序一致
	// shotInfo.Index 是按照分镜顺序从 1 开始递增的（前 3 个分镜合并成一个视频，sequence=1）
	sequence := shotInfo.Index

	// videoPrompt 已经在前面（第 571 行）构建好了，这里直接使用

	// 获取章节信息以获取 novel_id
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	videoEntity := &novel.Video{
		ID:          videoID,
		ChapterID:  chapterID,
		NarrationID: narration.ID,
		NovelID:    chapter.NovelID,
		UserID:     narration.UserID,
		Sequence:   sequence,
		VideoResourceID: uploadResult.ResourceID,
		Duration:        audioDuration,
		VideoType:       novel.VideoTypeNarration,
		Prompt:          videoPrompt,
		Version:         version,
		Status:          novel.VideoStatusCompleted,
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
	return s.GenerateFinalVideoForChapterWithVersion(ctx, chapterID, 0)
}

func (s *novelService) GenerateFinalVideoForChapterWithVersion(ctx context.Context, chapterID string, version int) (string, error) {
	return s.generateFinalVideoForChapter(ctx, chapterID, version)
}

func (s *novelService) generateFinalVideoForChapter(ctx context.Context, chapterID string, version int) (string, error) {
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return "", fmt.Errorf("find chapter: %w", err)
	}

	// 2. 确定要合并的版本号：version<=0 则取最新版本
	videoVersion, err := s.resolveVideoVersion(ctx, chapterID, version)
	if err != nil {
		return "", fmt.Errorf("resolve video version: %w", err)
	}

	// 2.5. 只获取指定版本的 narration 视频（确保只合并目标版本的视频）
	narrationVideos, err := s.videoRepo.FindByChapterIDAndVersion(ctx, chapterID, videoVersion)
	if err != nil {
		return "", fmt.Errorf("find narration videos for version %d: %w", videoVersion, err)
	}

	// 过滤出 narration_video 类型的视频
	var filteredNarrationVideos []*novel.Video
	for _, video := range narrationVideos {
		if video.VideoType == novel.VideoTypeNarration {
			filteredNarrationVideos = append(filteredNarrationVideos, video)
		}
	}

	if len(filteredNarrationVideos) == 0 {
		return "", fmt.Errorf("no narration videos found for chapter %s, version %d", chapterID, videoVersion)
	}

	// 按 sequence 排序
	sort.Slice(filteredNarrationVideos, func(i, j int) bool {
		return filteredNarrationVideos[i].Sequence < filteredNarrationVideos[j].Sequence
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
		Sequence:        1,
		VideoResourceID: uploadResult.ResourceID,
		Duration:        totalDuration,
		VideoType:       novel.VideoTypeFinal,
		Version:         videoVersion, // 使用与 narration 视频相同的版本号
		Status:          novel.VideoStatusCompleted,
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
func (s *novelService) GetVideosByStatus(ctx context.Context, status novel.VideoStatus) ([]*novel.Video, error) {
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
