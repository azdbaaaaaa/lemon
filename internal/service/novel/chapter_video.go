package novel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"

	"lemon/internal/model/novel"
	"lemon/internal/pkg/ark"
	"lemon/internal/pkg/ffmpeg"
	"lemon/internal/pkg/id"
	"lemon/internal/service"
)

// VideoService 章节视频服务接口
// 定义章节视频相关的能力
type VideoService interface {
	// GenerateFirstVideosForChapter 为章节的前两张图片生成视频（对应 gen_first_video_async.py）
	// 在 goroutine 中异步执行，自动使用最新版本号+1
	GenerateFirstVideosForChapter(ctx context.Context, chapterID string) ([]string, error)

	// GenerateNarrationVideosForChapter 为章节生成所有 narration 视频（对应 concat_narration_video.py）
	// 合并 narration 视频，添加 BGM 和音效
	GenerateNarrationVideosForChapter(ctx context.Context, chapterID string) ([]string, error)

	// GenerateFinalVideoForChapter 生成章节的最终完整视频（对应 concat_finish_video.py）
	// 拼接所有 narration 视频，添加 finish.mp4
	GenerateFinalVideoForChapter(ctx context.Context, chapterID string) (string, error)

	// GetVideoVersions 获取章节的所有视频版本号
	GetVideoVersions(ctx context.Context, chapterID string) ([]int, error)

	// GetVideosByStatus 根据状态查询视频（用于轮询）
	GetVideosByStatus(ctx context.Context, status string) ([]*novel.ChapterVideo, error)
}

// GenerateFirstVideosForChapter 为章节的前两张图片生成视频
// 对应 Python: gen_first_video_async.py
func (s *novelService) GenerateFirstVideosForChapter(ctx context.Context, chapterID string) ([]string, error) {
	// 1. 获取章节信息
	chapter, err := s.chapterRepo.FindByID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter: %w", err)
	}

	// 2. 获取章节的前两张图片（sequence=1 和 sequence=2）
	images, err := s.chapterImageRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter images: %w", err)
	}

	// 找到 sequence=1 和 sequence=2 的图片
	var firstImage, secondImage *novel.ChapterImage
	for _, img := range images {
		if img.Sequence == 1 {
			firstImage = img
		} else if img.Sequence == 2 {
			secondImage = img
		}
	}

	if firstImage == nil || secondImage == nil {
		return nil, fmt.Errorf("chapter must have at least 2 images (sequence 1 and 2)")
	}

	// 3. 获取对应的音频时长（sequence=1 和 sequence=2）
	audios, err := s.audioRepo.FindByChapterID(ctx, chapterID)
	if err != nil {
		return nil, fmt.Errorf("find chapter audios: %w", err)
	}

	var duration1, duration2 float64
	for _, audio := range audios {
		if audio.Sequence == 1 {
			duration1 = audio.Duration
		} else if audio.Sequence == 2 {
			duration2 = audio.Duration
		}
	}

	if duration1 == 0 {
		duration1 = 5 // 默认 5 秒
	}
	if duration2 == 0 {
		duration2 = 5 // 默认 5 秒
	}

	// 4. 自动生成下一个版本号
	videoVersion, err := s.getNextVideoVersion(ctx, chapterID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get next video version: %w", err)
	}

	// 5. 初始化 Ark 视频客户端
	videoConfig := ark.ArkVideoConfigFromEnv()
	arkVideoClient, err := ark.NewArkVideoClient(videoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ark video client: %w", err)
	}

	// 6. 在 goroutine 中异步生成两个视频
	var videoIDs []string
	video1ID := id.New()
	video2ID := id.New()

	// 创建视频记录（状态为 pending）
	video1Entity := &novel.ChapterVideo{
		ID:        video1ID,
		ChapterID: chapterID,
		UserID:    chapter.UserID,
		Sequence:  1,
		VideoType: "first_video",
		Version:   videoVersion,
		Status:    "pending",
	}
	if err := s.videoRepo.Create(ctx, video1Entity); err != nil {
		return nil, fmt.Errorf("failed to create video1 record: %w", err)
	}

	video2Entity := &novel.ChapterVideo{
		ID:        video2ID,
		ChapterID: chapterID,
		UserID:    chapter.UserID,
		Sequence:  2,
		VideoType: "first_video",
		Version:   videoVersion,
		Status:    "pending",
	}
	if err := s.videoRepo.Create(ctx, video2Entity); err != nil {
		return nil, fmt.Errorf("failed to create video2 record: %w", err)
	}

	videoIDs = append(videoIDs, video1ID, video2ID)

	// 在 goroutine 中异步生成视频
	go func() {
		// 添加 panic 恢复机制，确保即使 panic 也能记录错误
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("video1_id", video1ID).
					Str("video2_id", video2ID).
					Msg("生成视频时发生 panic")
				// 尝试更新状态为 failed
				ctx := context.Background()
				if err := s.videoRepo.UpdateStatus(ctx, video1ID, "failed", fmt.Sprintf("panic: %v", r)); err != nil {
					log.Error().Err(err).Str("video_id", video1ID).Msg("更新视频状态失败")
				}
				if err := s.videoRepo.UpdateStatus(ctx, video2ID, "failed", fmt.Sprintf("panic: %v", r)); err != nil {
					log.Error().Err(err).Str("video_id", video2ID).Msg("更新视频状态失败")
				}
			}
		}()

		// 验证参数
		if firstImage == nil {
			log.Error().Str("video_id", video1ID).Msg("firstImage 为 nil，无法生成视频")
			ctx := context.Background()
			if err := s.videoRepo.UpdateStatus(ctx, video1ID, "failed", "firstImage is nil"); err != nil {
				log.Error().Err(err).Str("video_id", video1ID).Msg("更新视频状态失败")
			}
			return
		}
		if secondImage == nil {
			log.Error().Str("video_id", video2ID).Msg("secondImage 为 nil，无法生成视频")
			ctx := context.Background()
			if err := s.videoRepo.UpdateStatus(ctx, video2ID, "failed", "secondImage is nil"); err != nil {
				log.Error().Err(err).Str("video_id", video2ID).Msg("更新视频状态失败")
			}
			return
		}
		if arkVideoClient == nil {
			log.Error().Str("video1_id", video1ID).Str("video2_id", video2ID).Msg("arkVideoClient 为 nil，无法生成视频")
			ctx := context.Background()
			if err := s.videoRepo.UpdateStatus(ctx, video1ID, "failed", "arkVideoClient is nil"); err != nil {
				log.Error().Err(err).Str("video_id", video1ID).Msg("更新视频状态失败")
			}
			if err := s.videoRepo.UpdateStatus(ctx, video2ID, "failed", "arkVideoClient is nil"); err != nil {
				log.Error().Err(err).Str("video_id", video2ID).Msg("更新视频状态失败")
			}
			return
		}

		log.Info().
			Str("video1_id", video1ID).
			Str("video2_id", video2ID).
			Msg("开始异步生成视频")

		// 生成第一个视频
		if err := s.generateSingleFirstVideo(context.Background(), video1ID, firstImage, int(duration1), arkVideoClient, chapter.UserID); err != nil {
			log.Error().Err(err).Str("video_id", video1ID).Msg("生成第一个视频失败")
		}

		// 生成第二个视频
		if err := s.generateSingleFirstVideo(context.Background(), video2ID, secondImage, int(duration2), arkVideoClient, chapter.UserID); err != nil {
			log.Error().Err(err).Str("video_id", video2ID).Msg("生成第二个视频失败")
		}
	}()

	return videoIDs, nil
}

// generateSingleFirstVideo 生成单个前两张图片的视频
func (s *novelService) generateSingleFirstVideo(
	ctx context.Context,
	videoID string,
	image *novel.ChapterImage,
	duration int,
	arkVideoClient *ark.ArkVideoClient,
	userID string,
) error {
	log.Info().
		Str("video_id", videoID).
		Str("image_resource_id", image.ImageResourceID).
		Int("duration", duration).
		Msg("开始生成视频")

	// 更新状态为 processing
	if err := s.videoRepo.UpdateStatus(ctx, videoID, "processing", ""); err != nil {
		log.Error().Err(err).Str("video_id", videoID).Msg("更新状态为 processing 失败")
		return fmt.Errorf("update status to processing: %w", err)
	}

	log.Info().Str("video_id", videoID).Msg("状态已更新为 processing")

	// 1. 从 resource 模块下载图片
	downloadReq := &service.DownloadFileRequest{
		ResourceID: image.ImageResourceID,
		UserID:     userID,
	}

	downloadResult, err := s.resourceService.DownloadFile(ctx, downloadReq)
	if err != nil {
		s.videoRepo.UpdateStatus(ctx, videoID, "failed", fmt.Sprintf("failed to download image: %v", err))
		return fmt.Errorf("download image: %w", err)
	}

	// 2. 读取图片数据
	imageData, err := io.ReadAll(downloadResult.Data)
	if err != nil {
		s.videoRepo.UpdateStatus(ctx, videoID, "failed", fmt.Sprintf("failed to read image data: %v", err))
		return fmt.Errorf("read image data: %w", err)
	}
	defer downloadResult.Data.Close()

	// 3. 将图片转换为 base64 data URL
	imageDataURL := ark.ConvertImageToDataURL(imageData, "image/jpeg")

	// 4. 调用 Ark API 生成视频（内部会轮询等待）
	prompt := "画面有明显的动态效果，动作大一些"
	videoData, err := arkVideoClient.GenerateVideoFromImage(ctx, imageDataURL, duration, prompt)
	if err != nil {
		s.videoRepo.UpdateStatus(ctx, videoID, "failed", fmt.Sprintf("Ark API failed: %v", err))
		return fmt.Errorf("generate video from image: %w", err)
	}

	// 5. 上传视频到 resource 模块
	fileName := fmt.Sprintf("%s_video_%d.mp4", image.ChapterID, image.Sequence)
	contentType := "video/mp4"
	ext := "mp4"

	uploadReq := &service.UploadFileRequest{
		UserID:      userID,
		FileName:    fileName,
		ContentType: contentType,
		Ext:         ext,
		Data:        bytes.NewReader(videoData),
	}

	uploadResult, err := s.resourceService.UploadFile(ctx, uploadReq)
	if err != nil {
		s.videoRepo.UpdateStatus(ctx, videoID, "failed", fmt.Sprintf("failed to upload video: %v", err))
		return fmt.Errorf("upload video file: %w", err)
	}

	// 6. 更新视频记录
	if err := s.videoRepo.UpdateVideoResourceID(ctx, videoID, uploadResult.ResourceID, float64(duration), prompt); err != nil {
		return fmt.Errorf("update video resource ID: %w", err)
	}

	if err := s.videoRepo.UpdateStatus(ctx, videoID, "completed", ""); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	log.Info().
		Str("video_id", videoID).
		Str("resource_id", uploadResult.ResourceID).
		Int("duration", duration).
		Msg("第一个视频生成成功")

	return nil
}

// GenerateNarrationVideosForChapter 为章节生成所有 narration 视频
// 对应 Python: concat_narration_video.py
// 逻辑：
//   - 从 ChapterNarration.Content.Scenes[].Shots[] 中提取所有 Shots
//   - 按照顺序编号为 narration_01, narration_02, narration_03, ...
//   - narration_01-03: 合并成一个视频（前10秒使用 first_video，后面用 image_01-03 平均分配）
//   - narration_04-30: 每个单独生成视频（静态图片转视频）
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

	// 6. 获取前两张图片的视频（first_video）
	firstVideos, err := s.videoRepo.FindByChapterIDAndType(ctx, chapterID, "first_video")
	if err != nil {
		return nil, fmt.Errorf("find first videos: %w", err)
	}

	var video1 *novel.ChapterVideo
	for _, v := range firstVideos {
		if v.Sequence == 1 && v.Status == "completed" {
			video1 = v
			break
		}
	}

	var videoIDs []string

	// 7. 处理 narration_01-03：合并成一个视频
	if len(allShots) >= 3 {
		mergedVideoID, err := s.generateMergedNarrationVideo(ctx, chapterID, narration, allShots[0:3], video1, videoVersion, ffmpegClient)
		if err != nil {
			log.Error().Err(err).Msg("生成合并 narration_01-03 视频失败")
		} else {
			videoIDs = append(videoIDs, mergedVideoID)
		}
	}

	// 8. 处理 narration_04-30：每个单独生成视频
	if len(allShots) > 3 {
		for i := 3; i < len(allShots) && i < 30; i++ {
			shotInfo := allShots[i]
			narrationNum := fmt.Sprintf("%02d", shotInfo.Index)

			videoID, err := s.generateSingleNarrationVideo(ctx, chapterID, narration, shotInfo, narrationNum, videoVersion, ffmpegClient)
			if err != nil {
				log.Error().Err(err).Str("narration_num", narrationNum).Msg("生成 narration 视频失败")
				continue
			}
			videoIDs = append(videoIDs, videoID)
		}
	}

	return videoIDs, nil
}

// generateNarration01Video 生成 narration_01 视频（使用 video_1 作为基础）
func (s *novelService) generateNarration01Video(
	ctx context.Context,
	narration *novel.ChapterNarration,
	video1 *novel.ChapterVideo,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	// 1. 获取 narration_01 的音频时长
	audios, err := s.audioRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find audios: %w", err)
	}

	var audioDuration float64
	for _, audio := range audios {
		audioDuration += audio.Duration
	}

	if audioDuration == 0 {
		return "", fmt.Errorf("audio duration is 0")
	}

	// 2. 下载 video_1
	downloadReq := &service.DownloadFileRequest{
		ResourceID: video1.VideoResourceID,
		UserID:     narration.UserID,
	}
	video1Result, err := s.resourceService.DownloadFile(ctx, downloadReq)
	if err != nil {
		return "", fmt.Errorf("download video_1: %w", err)
	}
	defer video1Result.Data.Close()

	// 3. 创建临时文件
	tmpDir := os.TempDir()
	tmpVideo1Path := filepath.Join(tmpDir, fmt.Sprintf("video1_%s.mp4", id.New()))
	defer os.Remove(tmpVideo1Path)

	// 保存 video_1 到临时文件
	video1File, err := os.Create(tmpVideo1Path)
	if err != nil {
		return "", fmt.Errorf("create temp video file: %w", err)
	}
	if _, err := io.Copy(video1File, video1Result.Data); err != nil {
		video1File.Close()
		return "", fmt.Errorf("copy video data: %w", err)
	}
	video1File.Close()

	// 4. 裁剪 video_1 到音频时长
	tmpCroppedPath := filepath.Join(tmpDir, fmt.Sprintf("video1_cropped_%s.mp4", id.New()))
	defer os.Remove(tmpCroppedPath)

	if err := ffmpegClient.CropVideo(ctx, tmpVideo1Path, tmpCroppedPath, audioDuration); err != nil {
		return "", fmt.Errorf("crop video: %w", err)
	}

	// 5. 获取字幕文件
	subtitle, err := s.subtitleRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find subtitle: %w", err)
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

	// 6. 获取音频文件
	audioDownloadReq := &service.DownloadFileRequest{
		ResourceID: audios[0].AudioResourceID,
		UserID:     narration.UserID,
	}
	audioResult, err := s.resourceService.DownloadFile(ctx, audioDownloadReq)
	if err != nil {
		return "", fmt.Errorf("download audio: %w", err)
	}
	defer audioResult.Data.Close()

	// 保存音频到临时文件
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

	// 7. 添加字幕到视频
	tmpWithSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("video_subtitle_%s.mp4", id.New()))
	defer os.Remove(tmpWithSubtitlePath)

	if err := ffmpegClient.AddSubtitles(ctx, tmpCroppedPath, tmpSubtitlePath, tmpWithSubtitlePath); err != nil {
		return "", fmt.Errorf("add subtitles: %w", err)
	}

	// 8. 混合音频（视频音频 + narration 音频）
	tmpFinalPath := filepath.Join(tmpDir, fmt.Sprintf("video_final_%s.mp4", id.New()))
	defer os.Remove(tmpFinalPath)

	// 简化：直接替换音频（不使用 BGM 和音效，第一版可简化）
	// TODO: 后续添加 BGM 和音效支持
	if err := s.replaceVideoAudio(ctx, tmpWithSubtitlePath, tmpAudioPath, tmpFinalPath, ffmpegClient); err != nil {
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

	fileName := fmt.Sprintf("%s_narration_01_video.mp4", narration.ChapterID)
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
	videoEntity := &novel.ChapterVideo{
		ID:              videoID,
		ChapterID:       narration.ChapterID,
		NarrationID:     narration.ID,
		UserID:          narration.UserID,
		Sequence:        1,
		VideoResourceID: uploadResult.ResourceID,
		Duration:        audioDuration,
		VideoType:       "narration_video",
		Version:         version,
		Status:          "completed",
	}

	if err := s.videoRepo.Create(ctx, videoEntity); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	return videoID, nil
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

// generateMergedNarrationVideo 生成合并的 narration_01-03 视频
// 对应 Python: create_merged_narration_video()
// 逻辑：
//   - 前10秒使用 first_video（video_1）
//   - 后面时间用 image_01-03 平均分配（静态图片转视频）
//   - 合并 ASS 和 MP3 文件
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
	video1 *novel.ChapterVideo,
	version int,
	ffmpegClient *ffmpeg.Client,
) (string, error) {
	if len(shots) != 3 {
		return "", fmt.Errorf("merged narration video requires exactly 3 shots, got %d", len(shots))
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
		totalAudioDuration += audios[i].Duration
	}

	// 2. 获取字幕文件（整体字幕，不是分段的）
	subtitle, err := s.subtitleRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find subtitle: %w", err)
	}

	// 3. 创建视频片段列表
	tmpDir := os.TempDir()
	var videoSegmentPaths []string
	defer func() {
		for _, path := range videoSegmentPaths {
			os.Remove(path)
		}
	}()

	// 3.1 前10秒使用 first_video（video_1）
	if video1 != nil {
		downloadReq := &service.DownloadFileRequest{
			ResourceID: video1.VideoResourceID,
			UserID:     narration.UserID,
		}
		video1Result, err := s.resourceService.DownloadFile(ctx, downloadReq)
		if err != nil {
			return "", fmt.Errorf("download video_1: %w", err)
		}
		defer video1Result.Data.Close()

		tmpVideo1Path := filepath.Join(tmpDir, fmt.Sprintf("video1_%s.mp4", id.New()))
		video1File, err := os.Create(tmpVideo1Path)
		if err != nil {
			return "", fmt.Errorf("create temp video file: %w", err)
		}
		if _, err := io.Copy(video1File, video1Result.Data); err != nil {
			video1File.Close()
			return "", fmt.Errorf("copy video data: %w", err)
		}
		video1File.Close()

		// 裁剪到10秒或总时长（取较小值）
		firstVideoDuration := 10.0
		if totalAudioDuration < 10.0 {
			firstVideoDuration = totalAudioDuration
		}

		tmpCroppedVideo1Path := filepath.Join(tmpDir, fmt.Sprintf("video1_cropped_%s.mp4", id.New()))
		if err := ffmpegClient.CropVideo(ctx, tmpVideo1Path, tmpCroppedVideo1Path, firstVideoDuration); err != nil {
			return "", fmt.Errorf("crop video_1: %w", err)
		}
		videoSegmentPaths = append(videoSegmentPaths, tmpCroppedVideo1Path)
		os.Remove(tmpVideo1Path) // 清理原始文件
	}

	// 3.2 后面时间用 image_01-03 平均分配（静态图片转视频）
	if totalAudioDuration > 10.0 {
		remainingDuration := totalAudioDuration - 10.0
		imageDurationEach := remainingDuration / 3.0

		for i, shotInfo := range shots {
			// 根据 scene_number 和 shot_number 查找图片
			image, err := s.chapterImageRepo.FindBySceneAndShot(ctx, chapterID, shotInfo.SceneNumber, shotInfo.ShotNumber)
			if err != nil {
				log.Warn().Err(err).Str("scene", shotInfo.SceneNumber).Str("shot", shotInfo.ShotNumber).Msg("图片未找到，跳过")
				continue
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

			// 从图片创建视频
			tmpImageVideoPath := filepath.Join(tmpDir, fmt.Sprintf("image_video_%d_%s.mp4", i+1, id.New()))
			if err := ffmpegClient.CreateImageVideo(ctx, tmpImagePath, tmpImageVideoPath, imageDurationEach, 720, 1280, 30); err != nil {
				return "", fmt.Errorf("create image video %d: %w", i+1, err)
			}
			videoSegmentPaths = append(videoSegmentPaths, tmpImageVideoPath)
			os.Remove(tmpImagePath) // 清理图片文件
		}
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

	// 5. 下载字幕文件
	subtitleDownloadReq := &service.DownloadFileRequest{
		ResourceID: subtitle.SubtitleResourceID,
		UserID:     narration.UserID,
	}
	subtitleResult, err := s.resourceService.DownloadFile(ctx, subtitleDownloadReq)
	if err != nil {
		return "", fmt.Errorf("download subtitle: %w", err)
	}
	defer subtitleResult.Data.Close()

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

	// 6. 合并前三个音频文件
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

	if err := ffmpegClient.AddSubtitles(ctx, tmpMergedVideoPath, tmpSubtitlePath, tmpWithSubtitlePath); err != nil {
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
	videoEntity := &novel.ChapterVideo{
		ID:              videoID,
		ChapterID:       chapterID,
		NarrationID:     narration.ID,
		UserID:          narration.UserID,
		Sequence:        1, // 合并视频的 sequence 为 1
		VideoResourceID: uploadResult.ResourceID,
		Duration:        totalAudioDuration,
		VideoType:       "narration_video",
		Version:         version,
		Status:          "completed",
	}

	if err := s.videoRepo.Create(ctx, videoEntity); err != nil {
		return "", fmt.Errorf("create video record: %w", err)
	}

	return videoID, nil
}

// generateSingleNarrationVideo 生成单个 narration 视频（narration_04-30，静态图片转视频）
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
	if audioDuration == 0 {
		return "", fmt.Errorf("audio duration is 0")
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

	// 4. 从图片创建视频（带 Ken Burns 效果）
	tmpVideoPath := filepath.Join(tmpDir, fmt.Sprintf("video_%s.mp4", id.New()))
	defer os.Remove(tmpVideoPath)

	if err := ffmpegClient.CreateImageVideo(ctx, tmpImagePath, tmpVideoPath, audioDuration, 720, 1280, 30); err != nil {
		return "", fmt.Errorf("create image video: %w", err)
	}

	// 5. 获取字幕文件（整体字幕，不是分段的）
	subtitle, err := s.subtitleRepo.FindByNarrationID(ctx, narration.ID)
	if err != nil {
		return "", fmt.Errorf("find subtitle: %w", err)
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

	// 6. 添加字幕到视频
	tmpWithSubtitlePath := filepath.Join(tmpDir, fmt.Sprintf("video_subtitle_%s.mp4", id.New()))
	defer os.Remove(tmpWithSubtitlePath)

	if err := ffmpegClient.AddSubtitles(ctx, tmpVideoPath, tmpSubtitlePath, tmpWithSubtitlePath); err != nil {
		return "", fmt.Errorf("add subtitles: %w", err)
	}

	// 7. 下载音频文件
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

	// 10. 上传视频
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

	// 11. 创建视频记录
	videoID := id.New()
	// 获取当前已生成的视频数量（用于 sequence）
	existingVideos, _ := s.videoRepo.FindByChapterIDAndType(ctx, chapterID, "narration_video")
	sequence := len(existingVideos) + 1

	videoEntity := &novel.ChapterVideo{
		ID:              videoID,
		ChapterID:       chapterID,
		NarrationID:     narration.ID,
		UserID:          narration.UserID,
		Sequence:        sequence,
		VideoResourceID: uploadResult.ResourceID,
		Duration:        audioDuration,
		VideoType:       "narration_video",
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
