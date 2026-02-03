package noveltools

import (
	"context"
	"time"
)

// LLMProvider 定义了调用大模型的接口
// 具体的「如何调用大模型」由调用方通过实现此接口注入，方便单测和替换实现
type LLMProvider interface {
	// Generate 根据提示词生成文本
	//
	// Args:
	//   - ctx: 上下文
	//   - prompt: 提示词
	//
	// Returns:
	//   - text: 生成的文本
	//   - err: 错误信息
	Generate(ctx context.Context, prompt string) (string, error)
}

// TTSProvider TTS提供者接口（用于单测/替换实现）
// 参考 Python 脚本 gen_audio.py 的 VoiceGenerator.generate_voice_with_timestamps
type TTSProvider interface {
	// GenerateVoiceWithTimestamps 生成语音并获取时间戳
	// 返回音频数据和时长，不保存到文件
	//
	// Args:
	//   - ctx: 上下文
	//   - text: 要转换的文本
	//   - speedRatio: 语速比例（默认1.0，1.2表示1.2倍速）
	//
	// Returns:
	//   - result: 生成结果（包含音频数据和时长）
	//   - err: 错误信息
	GenerateVoiceWithTimestamps(
		ctx context.Context,
		text string,
		speedRatio float64,
	) (*TTSResult, error)
}

// ImageProvider 图片生成提供者接口
// 统一抽象 T2P 和 ComfyUI 两种图片生成方式
type ImageProvider interface {
	// GenerateImage 生成图片
	// Args:
	//   - ctx: 上下文
	//   - prompt: 图片描述文本
	//   - filename: 输出文件名（用于标识，如 "chapter_001_image_01.jpeg"）
	// Returns:
	//   - imageData: 图片二进制数据
	//   - error: 错误信息
	GenerateImage(ctx context.Context, prompt, filename string) ([]byte, error)
}

// TTSResult TTS生成结果
type TTSResult struct {
	Success       bool           `json:"success"`        // 是否成功
	AudioData     []byte         `json:"-"`              // 音频数据（二进制，不序列化到 JSON）
	Duration      float64        `json:"duration"`       // 音频时长（秒）
	TimestampData *TimestampData `json:"timestamp_data"` // 时间戳数据
	ErrorMessage  string         `json:"error_message"`  // 错误信息
}

// TimestampData 时间戳数据
type TimestampData struct {
	Text                string          `json:"text"`                 // 原始文本
	Duration            float64         `json:"duration"`             // 音频时长（秒）
	CharacterTimestamps []CharTimestamp `json:"character_timestamps"` // 字符级时间戳
	GeneratedAt         time.Time       `json:"generated_at"`         // 生成时间
}

// CharTimestamp 字符时间戳
type CharTimestamp struct {
	Character string  `json:"character"`  // 字符
	StartTime float64 `json:"start_time"` // 开始时间（秒）
	EndTime   float64 `json:"end_time"`   // 结束时间（秒）
}
