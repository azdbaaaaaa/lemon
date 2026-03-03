package novel

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"lemon/internal/config"
	"lemon/internal/pkg/mongodb"
	"lemon/internal/pkg/storage"
	"lemon/internal/pkg/storagefactory"
	"lemon/internal/service"
	novelservice "lemon/internal/service/novel"
)

// cliEnv 小说 CLI 环境
// 封装了小说相关命令所需的依赖（MongoDB、Storage、Service 等）
type cliEnv struct {
	db           *mongodb.Client
	storage      storage.Storage
	resourceSvc  service.ResourceService
	novelSvc     novelservice.NovelService
	baseCtx      context.Context
	cleanupMongo func(context.Context)
}

// newCLIEnv 创建小说 CLI 环境
// 基于当前 viper 配置初始化 MongoDB、Storage、ResourceService 和 NovelService
func newCLIEnv() (*cliEnv, error) {
	cfg := &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
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

	env := &cliEnv{
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
func (e *cliEnv) Close() {
	e.cleanupMongo(e.baseCtx)
}
