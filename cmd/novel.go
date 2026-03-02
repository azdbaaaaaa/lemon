package cmd

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"lemon/internal/pkg/mongodb"
	"lemon/internal/pkg/storage"
	"lemon/internal/pkg/storagefactory"
	"lemon/internal/service"
	novelservice "lemon/internal/service/novel"
)

var novelCmd = &cobra.Command{
	Use:   "novel",
	Short: "小说相关工具命令",
	Long:  "提供上传小说文件、解析小说并切分章节等命令。",
}

func init() {
	rootCmd.AddCommand(novelCmd)
	novelCmd.AddCommand(novelUploadCmd)
	novelCmd.AddCommand(novelParseCmd)
}

// novelCLIEnv 小说 CLI 环境
// 封装了小说相关命令所需的依赖（MongoDB、Storage、Service 等）
type novelCLIEnv struct {
	db           *mongodb.Client
	storage      storage.Storage
	resourceSvc  service.ResourceService
	novelSvc     novelservice.NovelService
	baseCtx      context.Context
	cleanupMongo func(context.Context)
}

// newNovelCLIEnv 创建小说 CLI 环境
// 初始化 MongoDB、Storage、ResourceService 和 NovelService
func newNovelCLIEnv() (*novelCLIEnv, error) {
	cfg := GetConfig()
	if cfg == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}

	if cfg.Mongo.URI == "" {
		return nil, fmt.Errorf("mongo.uri 未配置，无法使用 novel 子命令")
	}

	ctx := context.Background()

	mongoClient, err := mongodb.New(&cfg.Mongo)
	if err != nil {
		return nil, fmt.Errorf("连接 MongoDB 失败: %w", err)
	}

	st, err := storagefactory.NewStorage(ctx, &cfg.Storage)
	if err != nil {
		_ = mongoClient.Close(context.Background())
		return nil, fmt.Errorf("初始化存储失败: %w", err)
	}

	// CLI 工具场景下不需要通过 API 生成下载 URL，这里传空字符串即可
	resourceSvc := service.NewResourceService(mongoClient.Database(), st, "")

	novelSvc, err := novelservice.NewNovelService(mongoClient.Database(), resourceSvc)
	if err != nil {
		_ = mongoClient.Close(context.Background())
		return nil, fmt.Errorf("初始化 NovelService 失败: %w", err)
	}

	env := &novelCLIEnv{
		db:          mongoClient,
		storage:     st,
		resourceSvc: resourceSvc,
		novelSvc:    novelSvc,
		baseCtx:     ctx,
		cleanupMongo: func(c context.Context) {
			if err := mongoClient.Close(c); err != nil {
				log.Warn().Err(err).Msg("关闭 MongoDB 连接失败")
			}
		},
	}

	return env, nil
}

// Close 关闭环境资源
func (e *novelCLIEnv) Close() {
	e.cleanupMongo(e.baseCtx)
}
