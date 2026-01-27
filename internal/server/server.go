package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"lemon/internal/config"
	"lemon/internal/handler"
	authHandler "lemon/internal/handler/auth"
	"lemon/internal/pkg/cache"
	"lemon/internal/pkg/mongodb"
	authRepo "lemon/internal/repository/auth"
	"lemon/internal/server/middleware"
	"lemon/internal/service"
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
			// TODO: 添加认证中间件
			// auth := v1.Group("")
			// auth.Use(middleware.Auth())
			{
				v1.POST("/auth/logout", authHdl.Logout)
				v1.GET("/auth/me", authHdl.GetMe)
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
