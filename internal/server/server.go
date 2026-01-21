package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"lemon/internal/config"
	"lemon/internal/handler"
	"lemon/internal/pkg/cache"
	"lemon/internal/pkg/mongodb"
	"lemon/internal/server/middleware"
	"lemon/internal/service"
)

// Server HTTP 服务器
type Server struct {
	cfg          *config.Config
	engine       *gin.Engine
	mongo        *mongodb.Client
	redis        *cache.RedisCache
	transformSvc *service.TransformService
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
	var transformSvc *service.TransformService
	if cfg.AI.APIKey != "" {
		ts, err := service.NewTransformService(context.Background(), &cfg.AI)
		if err != nil {
			log.Warn().Err(err).Msg("failed to initialize TransformService, continuing without it")
		} else {
			transformSvc = ts
			log.Info().Str("provider", cfg.AI.Provider).Str("model", cfg.AI.Model).Msg("initialized TransformService")
		}
	}

	srv := &Server{
		cfg:          cfg,
		engine:       engine,
		mongo:        mongoClient,
		redis:        redisCache,
		transformSvc: transformSvc,
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

	// API v1
	v1 := s.engine.Group("/api/v1")
	{
		// Chat 接口
		chatHandler := handler.NewChatHandler()
		v1.POST("/chat", chatHandler.Chat)
		v1.POST("/chat/stream", chatHandler.ChatStream)

		// Transform 接口 (文本转换)
		if s.transformSvc != nil {
			transformHandler := handler.NewTransformHandler(s.transformSvc)
			v1.POST("/transform", transformHandler.Transform)
		}

		// Conversation 接口
		convHandler := handler.NewConversationHandler(s.mongo, s.redis)
		v1.POST("/conversations", convHandler.Create)
		v1.GET("/conversations", convHandler.List)
		v1.GET("/conversations/:id", convHandler.Get)
		v1.DELETE("/conversations/:id", convHandler.Delete)
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
