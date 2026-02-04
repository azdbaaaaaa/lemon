package novel

// TaskStatus 任务状态（用于 Narration, Scene, Shot, Audio, Image, Subtitle）
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待处理
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
)

// String 返回状态的字符串表示
func (s TaskStatus) String() string {
	return string(s)
}

// VideoStatus 视频状态（包含 processing 状态）
type VideoStatus string

const (
	VideoStatusPending    VideoStatus = "pending"    // 待处理
	VideoStatusProcessing VideoStatus = "processing" // 处理中
	VideoStatusCompleted  VideoStatus = "completed"  // 已完成
	VideoStatusFailed     VideoStatus = "failed"     // 失败
)

// String 返回状态的字符串表示
func (s VideoStatus) String() string {
	return string(s)
}

// VideoType 视频类型
type VideoType string

const (
	VideoTypeNarration VideoType = "narration_video" // 解说视频
	VideoTypeFinal     VideoType = "final_video"     // 最终完整视频
)

// String 返回类型的字符串表示
func (t VideoType) String() string {
	return string(t)
}

// SubtitleFormat 字幕格式
type SubtitleFormat string

const (
	SubtitleFormatASS SubtitleFormat = "ass" // ASS 格式
	SubtitleFormatSRT SubtitleFormat = "srt" // SRT 格式
	SubtitleFormatVTT SubtitleFormat = "vtt" // VTT 格式
)

// String 返回格式的字符串表示
func (f SubtitleFormat) String() string {
	return string(f)
}
