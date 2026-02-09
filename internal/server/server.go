package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"lemon/internal/config"
	"lemon/internal/handler"
	authHandler "lemon/internal/handler/auth"
	novelHandler "lemon/internal/handler/novel"
	resourceHandler "lemon/internal/handler/resource"
	"lemon/internal/pkg/cache"
	"lemon/internal/pkg/jwt"
	"lemon/internal/pkg/mongodb"
	"lemon/internal/pkg/storagefactory"
	authRepo "lemon/internal/repository/auth"
	novelRepo "lemon/internal/repository/novel"
	"lemon/internal/server/middleware"
	"lemon/internal/service"
	novelService "lemon/internal/service/novel"
)

// Server HTTP 服务器
type Server struct {
	cfg    *config.Config
	engine *gin.Engine
	mongo  *mongodb.Client
	redis  *cache.RedisCache
	// transformSvc *service.TransformService // TODO: 修复transform service后启用
}

// New 创建服务器实例
func New(cfg *config.Config) (*Server, error) {
	// 设置 Gin 模式
	switch cfg.Server.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 引擎
	engine := gin.New()

	// 初始化 MongoDB (可选)
	var mongoClient *mongodb.Client
	if cfg.Mongo.URI != "" {
		client, err := mongodb.New(&cfg.Mongo)
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to MongoDB, continuing without it")
		} else {
			mongoClient = client
			log.Info().Str("database", cfg.Mongo.Database).Msg("connected to MongoDB")

			// 创建索引
			if err := mongodb.EnsureIndexes(mongoClient.Database()); err != nil {
				log.Warn().Err(err).Msg("failed to ensure indexes")
			}
		}
	}

	// 初始化 Redis (可选)
	var redisCache *cache.RedisCache
	if cfg.Redis.Addr != "" {
		rc, err := cache.NewRedisCache(&cfg.Redis)
		if err != nil {
			log.Warn().Err(err).Msg("failed to connect to Redis, continuing without it")
		} else {
			redisCache = rc
			log.Info().Str("addr", cfg.Redis.Addr).Msg("connected to Redis")
		}
	}

	// 初始化 TransformService (可选)
	// TODO: 修复transform service后启用
	// var transformSvc *service.TransformService
	// if cfg.AI.APIKey != "" {
	// 	ts, err := service.NewTransformService(context.Background(), &cfg.AI)
	// 	if err != nil {
	// 		log.Warn().Err(err).Msg("failed to initialize TransformService, continuing without it")
	// 	} else {
	// 		transformSvc = ts
	// 		log.Info().Str("provider", cfg.AI.Provider).Str("model", cfg.AI.Model).Msg("initialized TransformService")
	// 	}
	// }

	srv := &Server{
		cfg:    cfg,
		engine: engine,
		mongo:  mongoClient,
		redis:  redisCache,
		// transformSvc: transformSvc, // TODO: 修复transform service后启用
	}

	// 设置路由
	srv.setupRoutes()

	return srv, nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 全局中间件
	s.engine.Use(middleware.Recovery())
	s.engine.Use(middleware.RequestID())
	s.engine.Use(middleware.Logger())
	s.engine.Use(middleware.CORS())

	// 健康检查
	healthHandler := handler.NewHealthHandler()
	s.engine.GET("/health", healthHandler.Health)
	s.engine.GET("/ready", healthHandler.Ready)

	// Swagger 文档
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := s.engine.Group("/api/v1")
	{
		// 认证接口（公开）
		var jwtUtil *jwt.JWT
		if s.mongo != nil {
			userRepo := authRepo.NewUserRepo(s.mongo.Database())
			refreshTokenRepo := authRepo.NewRefreshTokenRepo(s.mongo.Database())

			// 从配置读取JWT参数，如果没有配置则使用默认值
			jwtSecret := s.cfg.Auth.JWTSecret
			if jwtSecret == "" {
				jwtSecret = "default-secret-key-change-in-production"
				log.Warn().Msg("JWT secret not configured, using default (NOT SECURE for production)")
			}

			accessTokenExpiry := s.cfg.Auth.AccessTokenExpiry
			if accessTokenExpiry == 0 {
				accessTokenExpiry = 24 * time.Hour
			}

			refreshTokenExpiry := s.cfg.Auth.RefreshTokenExpiry
			if refreshTokenExpiry == 0 {
				refreshTokenExpiry = 7 * 24 * time.Hour
			}

			// 创建 JWT 工具实例（用于认证中间件）
			jwtUtil = jwt.NewJWT(jwtSecret, accessTokenExpiry)

			authSvc := service.NewAuthService(
				userRepo,
				refreshTokenRepo,
				jwtSecret,
				accessTokenExpiry,
				refreshTokenExpiry,
			)
			authHdl := authHandler.NewHandler(authSvc)

			v1.POST("/auth/register", authHdl.Register)
			v1.POST("/auth/login", authHdl.Login)
			v1.POST("/auth/refresh", authHdl.Refresh)

			// 需要认证的接口
			authGroup := v1.Group("")
			authGroup.Use(middleware.Auth(jwtUtil))
			{
				authGroup.POST("/auth/logout", authHdl.Logout)
				authGroup.GET("/auth/me", authHdl.GetMe)
			}
		} else {
			log.Warn().Msg("MongoDB not configured, auth endpoints disabled")
		}

		// 用户管理接口（需要管理员权限）
		// TODO: 实现user handler（需要先完成model定义）
		// TODO: 添加权限中间件

		// Chat 接口
		// TODO: 实现Chat功能（需要先完成conversation模块的设计）

		// Transform 接口 (文本转换)
		// TODO: 修复transform handler（需要修复model引用）

		// Conversation 接口
		// TODO: 实现conversation模块（需要先完成model定义）

		// Resource 接口（资源管理）
		if s.mongo != nil && jwtUtil != nil {
			// 初始化 ResourceService（需要 storage）
			storage, err := storagefactory.NewStorage(context.Background(), &s.cfg.Storage)
			if err != nil {
				log.Warn().Err(err).Msg("failed to initialize storage, resource endpoints disabled")
			} else {
				// 构建API基础URL
				baseURL := s.buildBaseURL()
				resourceSvc := service.NewResourceService(s.mongo.Database(), storage, baseURL)
				resourceHdl := resourceHandler.NewHandler(resourceSvc)

				// 资源下载接口（公开访问，不需要认证）
				// 支持两种格式：/resources/:resource_id/download 和 /resources/:resource_id/download.:ext
				v1.GET("/resources/:resource_id/download", resourceHdl.DownloadFile)
				v1.GET("/resources/:resource_id/download.:ext", resourceHdl.DownloadFile)

				// 资源管理接口（需要认证）
				resourceGroup := v1.Group("")
				resourceGroup.Use(middleware.Auth(jwtUtil))
				{
					resourceGroup.POST("/resources/upload", resourceHdl.UploadFile)
					resourceGroup.GET("/resources", resourceHdl.ListResources)
					resourceGroup.GET("/resources/:resource_id", resourceHdl.GetResource)
					resourceGroup.GET("/resources/:resource_id/download-url", resourceHdl.GetDownloadURL)
				}
			}
		} else {
			if s.mongo == nil {
				log.Warn().Msg("MongoDB not configured, resource endpoints disabled")
			}
			if jwtUtil == nil {
				log.Warn().Msg("JWT not configured, resource endpoints disabled")
			}
		}

		// Novel 接口（小说与创作相关）
		if s.mongo != nil && jwtUtil != nil {
			// 初始化 ResourceService（需要 storage）
			storage, err := storagefactory.NewStorage(context.Background(), &s.cfg.Storage)
			if err != nil {
				log.Warn().Err(err).Msg("failed to initialize storage, novel endpoints disabled")
			} else {
				db := s.mongo.Database()
				// 构建API基础URL
				baseURL := s.buildBaseURL()
				resourceSvc := service.NewResourceService(db, storage, baseURL)

				// 初始化 NovelService
				novelSvc, err := novelService.NewNovelService(db, resourceSvc)
				if err != nil {
					log.Warn().Err(err).Msg("failed to initialize NovelService, novel endpoints disabled")
				} else {
					// 初始化所有 Repository（用于 ContentService）
					novelRepoInst := novelRepo.NewNovelRepo(db)
					chapterRepoInst := novelRepo.NewChapterRepo(db)
					sceneRepoInst := novelRepo.NewSceneRepo(db)
					shotRepoInst := novelRepo.NewShotRepo(db)
					characterRepoInst := novelRepo.NewCharacterRepo(db)
					propRepoInst := novelRepo.NewPropRepo(db)
					// 初始化 ContentService
					contentSvc := novelService.NewContentService(novelSvc, novelRepoInst, chapterRepoInst, sceneRepoInst, shotRepoInst, characterRepoInst, propRepoInst)
					// 创建Handler，同时传入NovelService、ContentService和ResourceService
					contentHdl := novelHandler.NewHandler(novelSvc, contentSvc, resourceSvc)

					// 内容管理接口（需要认证）
					contentGroup := v1.Group("")
					contentGroup.Use(middleware.Auth(jwtUtil))
					{
						// 剧本管理接口
						contentGroup.POST("/novels", contentHdl.CreateNovel)
						contentGroup.GET("/novels", contentHdl.ListNovels)
						contentGroup.GET("/novels/detail", contentHdl.GetNovel)
						contentGroup.POST("/novels/generate-content", contentHdl.GenerateContent)
						contentGroup.GET("/novels/generation-status", contentHdl.GetGenerationStatus)

						// 章节管理接口
						contentGroup.GET("/chapters", contentHdl.GetChapters)
						contentGroup.POST("/chapters/split", contentHdl.SplitChapters)

						// 场景管理接口
						contentGroup.GET("/scenes", contentHdl.GetScenes)
						contentGroup.POST("/scenes/generate", contentHdl.GenerateScenes)
						contentGroup.PUT("/scenes/version", contentHdl.SetActiveSceneVersion)

						// 镜头管理接口
						contentGroup.GET("/shots", contentHdl.GetShots) // 支持 chapter_id（列表）或 shot_id（详情）
						contentGroup.PUT("/shots", contentHdl.UpdateShot)
						contentGroup.POST("/shots/images", contentHdl.GenerateShotImages)
						contentGroup.GET("/shots/images", contentHdl.GetShotImages)
						contentGroup.POST("/shots/audio", contentHdl.GenerateShotAudio)
						contentGroup.GET("/shots/audios", contentHdl.GetShotAudios)
						contentGroup.POST("/shots/video", contentHdl.GenerateShotVideo)
						contentGroup.GET("/shots/videos", contentHdl.GetShotVideos)

						// 角色管理接口
						contentGroup.GET("/characters", contentHdl.GetCharactersByNovelID)
						contentGroup.GET("/characters/detail", contentHdl.GetCharacterByName)
						contentGroup.POST("/characters/sync", contentHdl.SyncCharacters)
						contentGroup.POST("/characters/generate-from-novel", contentHdl.GenerateCharactersFromNovel)
						contentGroup.POST("/characters/images", contentHdl.GenerateCharacterImages)
						contentGroup.GET("/characters/images", contentHdl.GetCharacterImages)
						contentGroup.GET("/characters/images/status", contentHdl.GetCharacterImageGenerationStatus)

						// 道具管理接口
						contentGroup.GET("/props", contentHdl.GetPropsByNovelID)
						contentGroup.POST("/props/images", contentHdl.GeneratePropImages)
						contentGroup.GET("/props/images", contentHdl.GetPropImages)
						contentGroup.GET("/props/images/status", contentHdl.GetPropImageGenerationStatus)
					}

					// 其他接口（音频、字幕、视频、图片等，使用 contentHdl）
					// 注意：这些接口可能依赖已删除的narration模块，需要后续重构
					novelGroup := v1.Group("")
					novelGroup.Use(middleware.Auth(jwtUtil))
					{
						// 音频生成接口
						novelGroup.POST("/narrations/audios", contentHdl.GenerateAudios)
						novelGroup.GET("/narrations/audios", contentHdl.ListAudiosByNarration)
						novelGroup.GET("/narrations/audios/versions", contentHdl.GetAudioVersions)
						// 基于章节的音频生成接口（新接口）
						novelGroup.POST("/chapters/audios", contentHdl.GenerateAudiosForChapter)

						// 字幕生成接口
						novelGroup.POST("/narrations/subtitles", contentHdl.GenerateSubtitles)
						novelGroup.GET("/narrations/subtitles", contentHdl.ListSubtitlesByNarration)
						novelGroup.GET("/novels/chapters/subtitles/versions", contentHdl.GetSubtitleVersions)

						// 图片生成接口
						novelGroup.POST("/narrations/images", contentHdl.GenerateImages)
						novelGroup.GET("/narrations/images", contentHdl.ListImagesByNarration)
						novelGroup.GET("/novels/chapters/images/versions", contentHdl.GetImageVersions)

						// 视频生成接口
						novelGroup.POST("/chapters/videos/shots", contentHdl.GenerateShotVideos)
						novelGroup.POST("/chapters/videos", contentHdl.GenerateVideosForChapter)
						novelGroup.POST("/chapters/videos/final", contentHdl.GenerateFinalVideo)

						// 视频查询接口
						novelGroup.GET("/novels/chapters/videos", contentHdl.ListVideosByChapter)
						novelGroup.GET("/novels/chapters/videos/versions", contentHdl.GetVideoVersions)
						novelGroup.GET("/videos", contentHdl.GetVideosByStatus)
						// 基于章节的视频查询接口（新接口）
						novelGroup.GET("/chapters/videos", contentHdl.ListVideosByChapter)
						// 基于章节的音频查询接口（新接口）
						novelGroup.GET("/chapters/audios", contentHdl.ListAudiosByChapter)
					}
				}
			}
		} else {
			if s.mongo == nil {
				log.Warn().Msg("MongoDB not configured, novel endpoints disabled")
			}
			if jwtUtil == nil {
				log.Warn().Msg("JWT not configured, novel endpoints disabled")
			}
		}
	}
}

// Run 启动服务器
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
	}

	// 启动服务器
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// 等待关闭信号或错误
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down server...")

		// 关闭连接
		if s.mongo != nil {
			if err := s.mongo.Close(context.Background()); err != nil {
				log.Error().Err(err).Msg("failed to close MongoDB connection")
			}
		}
		if s.redis != nil {
			if err := s.redis.Close(); err != nil {
				log.Error().Err(err).Msg("failed to close Redis connection")
			}
		}

		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// Engine 获取 Gin 引擎 (用于测试)
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// buildBaseURL 构建API基础URL
func (s *Server) buildBaseURL() string {
	host := s.cfg.Server.Host
	if host == "" {
		host = "127.0.0.1"
	} else if host == "0.0.0.0" {
		// 将 0.0.0.0 转换为 127.0.0.1，以便生成可访问的URL
		host = "127.0.0.1"
	}
	port := s.cfg.Server.Port
	if port == 0 {
		port = 7080
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}
