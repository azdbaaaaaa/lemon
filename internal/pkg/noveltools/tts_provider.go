package noveltools

import (
	"context"
	"time"
)

// TimestampData 时间戳数据
type TimestampData struct {
	Text                string          `json:"text"`                 // 原始文本
	AudioFile           string          `json:"audio_file"`           // 音频文件路径
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

// TTSResult TTS生成结果
type TTSResult struct {
	Success       bool           `json:"success"`        // 是否成功
	AudioPath     string         `json:"audio_path"`     // 音频文件路径
	TimestampData *TimestampData `json:"timestamp_data"` // 时间戳数据
	ErrorMessage  string         `json:"error_message"`  // 错误信息
}

// TTSProvider TTS提供者接口（用于单测/替换实现）
// 参考 Python 脚本 gen_audio.py 的 VoiceGenerator.generate_voice_with_timestamps
type TTSProvider interface {
	// GenerateVoiceWithTimestamps 生成语音并获取时间戳
	//
	// Args:
	//   - ctx: 上下文
	//   - text: 要转换的文本
	//   - audioPath: 音频文件保存路径
	//   - speedRatio: 语速比例（默认1.0，1.2表示1.2倍速）
	//
	// Returns:
	//   - result: 生成结果
	//   - err: 错误信息
	GenerateVoiceWithTimestamps(
		ctx context.Context,
		text string,
		audioPath string,
		speedRatio float64,
	) (*TTSResult, error)
}
