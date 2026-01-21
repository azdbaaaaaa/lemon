package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"lemon/internal/config"
)

// Init 初始化全局日志
func Init(cfg *config.LogConfig) error {
	// 设置日志级别
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// 设置时间格式
	switch cfg.TimeFormat {
	case "Unix":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	case "UnixMs":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	default:
		zerolog.TimeFieldFormat = time.RFC3339
	}

	// 设置输出
	var output io.Writer = os.Stdout
	if cfg.Output == "file" && cfg.FilePath != "" {
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return err
		}
		output = file
	}

	// Console 格式 (开发环境友好)
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	}

	// 设置全局 logger
	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

	return nil
}

// Get 获取全局 logger
func Get() zerolog.Logger {
	return log.Logger
}
