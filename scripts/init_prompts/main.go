package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"lemon/internal/config"
	"lemon/internal/model/prompt"
	"lemon/internal/pkg/id"
	"lemon/internal/pkg/logger"
	"lemon/internal/pkg/mongodb"
	promptservice "lemon/internal/service/prompt"
)

// 基础模板定义：从 internal/prompts 目录加载的提示词
type basePromptFile struct {
	Type     string
	Code     string
	Language string
	File     string
	Title    string
}

var basePromptFiles = []basePromptFile{
	{
		Type:     "scene",
		Code:     "default",
		Language: "zh-CN",
		File:     "scene.md",
		Title:    "场景拆解（默认）",
	},
	{
		Type:     "shot",
		Code:     "default",
		Language: "zh-CN",
		File:     "shot.md",
		Title:    "镜头结构生成（默认）",
	},
	{
		Type:     "shot_video",
		Code:     "default",
		Language: "zh-CN",
		File:     "shot_video.md",
		Title:    "镜头视频与解说（默认）",
	},
	{
		Type:     "character_image",
		Code:     "default",
		Language: "en-US",
		File:     "character_image.md",
		Title:    "角色图像提示词（默认）",
	},
	{
		Type:     "character",
		Code:     "asset_extraction",
		Language: "zh-CN",
		File:     "character.md",
		Title:    "角色资产提取（整部小说）",
	},
}

func main() {
	// 1. 加载配置（与 cmd/root.go 保持一致的搜索路径）
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.lemon")

	viper.SetEnvPrefix("LEMON")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
		os.Exit(1)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	// 2. 连接 MongoDB
	client, err := mongodb.New(&cfg.Mongo)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect mongo")
	}
	defer func() {
		_ = client.Close(context.Background())
	}()

	db := client.Database()
	ctx := context.Background()

	// 3. 初始化 PromptService
	svc, err := promptservice.NewPromptService(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init PromptService")
	}

	// 4. 读取并写入基础模板
	baseDir := "./internal/prompts"
	var successCount int

	for _, item := range basePromptFiles {
		path := filepath.Join(baseDir, item.File)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Error().Err(err).Str("file", path).Msg("failed to read prompt file")
			continue
		}

		tpl := &prompt.PromptTemplate{
			ID:          id.New(),
			Type:        item.Type,
			Code:        item.Code,
			Language:    item.Language,
			Title:       item.Title,
			Description: "",
			Content:     string(content),
		}

		if _, err := svc.CreateTemplate(ctx, tpl); err != nil {
			log.Error().
				Err(err).
				Str("type", item.Type).
				Str("code", item.Code).
				Str("language", item.Language).
				Msg("failed to create prompt template")
			continue
		}

		successCount++
		log.Info().
			Str("type", item.Type).
			Str("code", item.Code).
			Str("language", item.Language).
			Str("file", path).
			Msg("prompt template initialized")
	}

	fmt.Printf("Prompt templates initialized: %d/%d\n", successCount, len(basePromptFiles))
}
