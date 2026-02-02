package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client FFmpeg 客户端
// 用于封装 FFmpeg 命令调用
type Client struct {
	ffmpegPath  string // FFmpeg 可执行文件路径（默认: ffmpeg）
	ffprobePath string // FFprobe 可执行文件路径（默认: ffprobe）
}

// NewClient 创建 FFmpeg 客户端
func NewClient() *Client {
	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	ffprobePath := os.Getenv("FFPROBE_PATH")
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}

	return &Client{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}
}

// VideoInfo 视频信息
type VideoInfo struct {
	Width    int     // 宽度
	Height   int     // 高度
	FPS      float64 // 帧率
	Duration float64 // 时长（秒）
}

// AudioInfo 音频信息
type AudioInfo struct {
	Duration float64 // 时长（秒）
}

// GetVideoInfo 获取视频信息
func (c *Client) GetVideoInfo(ctx context.Context, videoPath string) (*VideoInfo, error) {
	// 使用 ffprobe 获取视频信息
	// ffprobe -v error -select_streams v:0 -show_entries stream=width,height,r_frame_rate -show_entries format=duration -of json video.mp4
	cmd := exec.CommandContext(ctx, c.ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate",
		"-show_entries", "format=duration",
		"-of", "json",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// 解析 JSON 输出
	// 简化实现：直接解析关键字段
	// TODO: 使用完整的 JSON 解析
	outputStr := string(output)

	// 提取 width, height, duration, r_frame_rate
	// 这里使用简单的字符串解析，实际应该使用 JSON 解析库
	var info VideoInfo

	// 解析 width
	if idx := strings.Index(outputStr, `"width":`); idx != -1 {
		var width int
		if _, err := fmt.Sscanf(outputStr[idx:], `"width":%d`, &width); err == nil {
			info.Width = width
		}
	}

	// 解析 height
	if idx := strings.Index(outputStr, `"height":`); idx != -1 {
		var height int
		if _, err := fmt.Sscanf(outputStr[idx:], `"height":%d`, &height); err == nil {
			info.Height = height
		}
	}

	// 解析 duration
	if idx := strings.Index(outputStr, `"duration":`); idx != -1 {
		var duration float64
		if _, err := fmt.Sscanf(outputStr[idx:], `"duration":"%f"`, &duration); err == nil {
			info.Duration = duration
		}
	}

	// 解析 r_frame_rate (格式: "30000/1000")
	if idx := strings.Index(outputStr, `"r_frame_rate":`); idx != -1 {
		var num, den int
		if _, err := fmt.Sscanf(outputStr[idx:], `"r_frame_rate":"%d/%d"`, &num, &den); err == nil && den > 0 {
			info.FPS = float64(num) / float64(den)
		}
	}

	return &info, nil
}

// GetAudioInfo 获取音频信息
func (c *Client) GetAudioInfo(ctx context.Context, audioPath string) (*AudioInfo, error) {
	// 使用 ffprobe 获取音频信息
	cmd := exec.CommandContext(ctx, c.ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		audioPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// 解析 JSON 输出
	outputStr := string(output)
	var info AudioInfo

	// 解析 duration
	if idx := strings.Index(outputStr, `"duration":`); idx != -1 {
		var duration float64
		if _, err := fmt.Sscanf(outputStr[idx:], `"duration":"%f"`, &duration); err == nil {
			info.Duration = duration
		}
	}

	return &info, nil
}

// CreateImageVideo 从图片创建视频（带 Ken Burns 效果）
// 参考 Python: create_image_video_with_effects()
func (c *Client) CreateImageVideo(ctx context.Context, imagePath, outputPath string, duration float64, width, height int, fps int) error {
	// 定义 Ken Burns 动态效果（简化版本，随机选择一种）
	// 实际实现应该支持多种效果选择
	totalFrames := int(duration * float64(fps))

	// 简单的缩放效果
	zoomEffect := fmt.Sprintf("zoompan=z='min(1.0+on*0.0008,1.3)':x='iw/2-(iw/zoom/2)':y='ih/2-(ih/zoom/2)':d=%d:s=%dx%d:fps=%d",
		totalFrames, width, height, fps)

	// 构建 FFmpeg 命令
	// ffmpeg -y -loop 1 -i image.jpg -t duration -vf "scale=width:height:force_original_aspect_ratio=increase,crop=width:height,zoompan=..." -c:v libx264 -pix_fmt yuv420p -r fps output.mp4
	args := []string{
		"-y", // 覆盖输出文件
		"-loop", "1",
		"-i", imagePath,
		"-t", fmt.Sprintf("%.2f", duration),
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,%s",
			width, height, width, height, zoomEffect),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-r", fmt.Sprintf("%d", fps),
		outputPath,
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	cmd.Stderr = os.Stderr // 输出错误信息到 stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	log.Info().
		Str("image", imagePath).
		Str("output", outputPath).
		Float64("duration", duration).
		Msg("图片视频创建成功")

	return nil
}

// ConcatVideos 合并多个视频文件
// 使用 concat demuxer（需要创建 concat list 文件）
func (c *Client) ConcatVideos(ctx context.Context, videoPaths []string, outputPath string) error {
	if len(videoPaths) == 0 {
		return fmt.Errorf("no videos to concat")
	}

	// 创建临时目录
	tempDir := filepath.Dir(outputPath)
	concatListFile := filepath.Join(tempDir, fmt.Sprintf("concat_list_%d.txt", time.Now().Unix()))

	// 创建 concat list 文件
	file, err := os.Create(concatListFile)
	if err != nil {
		return fmt.Errorf("create concat list file: %w", err)
	}
	defer os.Remove(concatListFile) // 清理临时文件

	for _, videoPath := range videoPaths {
		absPath, err := filepath.Abs(videoPath)
		if err != nil {
			return fmt.Errorf("get absolute path: %w", err)
		}
		fmt.Fprintf(file, "file '%s'\n", absPath)
	}
	file.Close()

	// 构建 FFmpeg 命令
	// ffmpeg -f concat -safe 0 -i concat_list.txt -c copy output.mp4
	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatListFile,
		"-c", "copy", // 使用 copy 避免重新编码
		outputPath,
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat failed: %w", err)
	}

	log.Info().
		Int("count", len(videoPaths)).
		Str("output", outputPath).
		Msg("视频合并成功")

	return nil
}

// StandardizeVideo 标准化视频（分辨率、帧率）
func (c *Client) StandardizeVideo(ctx context.Context, inputPath, outputPath string, width, height int, fps int) error {
	// 构建视频滤镜
	// scale=width:height:force_original_aspect_ratio=increase,crop=width:height:(in_w-width)/2:(in_h-height)/2,setsar=1
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:(in_w-%d)/2:(in_h-%d)/2,setsar=1",
		width, height, width, height, width, height)

	args := []string{
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a?", // 可选音频流
		"-vf", vf,
		"-r", fmt.Sprintf("%d", fps),
		"-c:v", "libx264",
		"-crf", "20",
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "160k",
		"-movflags", "+faststart",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg standardize failed: %w", err)
	}

	log.Info().
		Str("input", inputPath).
		Str("output", outputPath).
		Int("width", width).
		Int("height", height).
		Int("fps", fps).
		Msg("视频标准化成功")

	return nil
}

// AddSubtitles 添加字幕到视频（ASS 格式）
func (c *Client) AddSubtitles(ctx context.Context, videoPath, assPath, outputPath string) error {
	// 构建 FFmpeg 命令
	// ffmpeg -i video.mp4 -vf "ass=subtitle.ass" output.mp4
	args := []string{
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf("ass=%s", assPath),
		"-c:v", "libx264",
		"-c:a", "copy",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg add subtitles failed: %w", err)
	}

	log.Info().
		Str("video", videoPath).
		Str("subtitle", assPath).
		Str("output", outputPath).
		Msg("字幕添加成功")

	return nil
}

// MixAudio 混合音频（视频音频 + BGM + 音效）
func (c *Client) MixAudio(ctx context.Context, videoPath string, bgmPath string, soundEffectPaths []string, outputPath string) error {
	// 构建复杂的音频滤镜
	// 使用 amix 或 amerge 混合多个音频流
	// 简化实现：先合并音效，再与 BGM 和视频音频混合

	// 构建输入参数
	args := []string{"-y"}

	// 添加视频输入
	args = append(args, "-i", videoPath)

	// 添加 BGM 输入（如果存在）
	if bgmPath != "" {
		args = append(args, "-i", bgmPath)
	}

	// 添加音效输入
	for _, sePath := range soundEffectPaths {
		if sePath != "" {
			args = append(args, "-i", sePath)
		}
	}

	// 构建音频滤镜
	// 简化实现：使用 amix 混合所有音频
	// 实际实现应该支持音量控制和淡入淡出
	filterComplex := ""
	if bgmPath != "" || len(soundEffectPaths) > 0 {
		// 构建 amix 输入列表
		inputs := []string{"0:a"} // 视频音频
		inputCount := 1

		if bgmPath != "" {
			inputs = append(inputs, fmt.Sprintf("%d:a", inputCount))
			inputCount++
		}

		for range soundEffectPaths {
			if soundEffectPaths[0] != "" {
				inputs = append(inputs, fmt.Sprintf("%d:a", inputCount))
				inputCount++
			}
		}

		// amix=inputs=3:duration=longest
		filterComplex = fmt.Sprintf("amix=inputs=%d:duration=longest", len(inputs))
	}

	if filterComplex != "" {
		args = append(args, "-filter_complex", filterComplex)
		args = append(args, "-c:v", "copy") // 视频流直接复制
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	} else {
		args = append(args, "-c", "copy") // 如果没有音频混合，直接复制
	}

	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg mix audio failed: %w", err)
	}

	log.Info().
		Str("video", videoPath).
		Str("bgm", bgmPath).
		Int("sound_effects", len(soundEffectPaths)).
		Str("output", outputPath).
		Msg("音频混合成功")

	return nil
}

// CropVideo 裁剪视频时长
func (c *Client) CropVideo(ctx context.Context, inputPath, outputPath string, duration float64) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-t", fmt.Sprintf("%.2f", duration),
		"-c", "copy", // 使用 copy 避免重新编码
		outputPath,
	}

	cmd := exec.CommandContext(ctx, c.ffmpegPath, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg crop failed: %w", err)
	}

	return nil
}
