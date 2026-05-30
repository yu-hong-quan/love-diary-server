// Package main 恋爱日记 API 服务入口。
// 负责加载配置、连接 PostgreSQL、注册路由，接口与 love-diary 前端及原 Node 服务保持一致。
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"love-diary-go/internal/config"
	"love-diary-go/internal/database"
	"love-diary-go/internal/handlers"
	"love-diary-go/internal/middleware"
	"love-diary-go/internal/repository"
	"love-diary-go/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// 从 .env / 环境变量加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置错误: %v", err)
	}

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer pool.Close()

	fileStore, err := storage.New(cfg.UploadDir)
	if err != nil {
		log.Fatalf("upload storage: %v", err)
	}
	log.Printf("uploads directory: %s (public URL prefix %s)", fileStore.Root(), storage.URLPrefix)

	// 数据访问层
	travelRepo := repository.NewTravelRepo(pool, fileStore)
	dailyRepo := repository.NewDailyRepo(pool, fileStore)
	whisperRepo := repository.NewWhisperRepo(pool)
	specialDateRepo := repository.NewSpecialDateRepo(pool)
	treeRepo := repository.NewTreeRepo(pool)
	userRepo := repository.NewUserRepo(pool)
	gameRepo := repository.NewGameRepo(pool)

	// HTTP 处理层
	authHandler := handlers.NewAuthHandler(userRepo, fileStore, cfg)
	travelHandler := handlers.NewTravelHandler(travelRepo)
	dailyHandler := handlers.NewDailyHandler(dailyRepo)
	travelCommentRepo := repository.NewTravelCommentRepo(pool, travelRepo)
	dailyCommentRepo := repository.NewDailyCommentRepo(pool, dailyRepo)
	travelCommentHandler := handlers.NewTravelCommentHandler(travelCommentRepo, travelRepo)
	dailyCommentHandler := handlers.NewDailyCommentHandler(dailyCommentRepo, dailyRepo)
	whisperHandler := handlers.NewWhisperHandler(whisperRepo)
	specialDateHandler := handlers.NewSpecialDateHandler(specialDateRepo)
	treeHandler := handlers.NewTreeHandler(treeRepo, cfg)
	uploadHandler := handlers.NewUploadHandler(fileStore, userRepo)
	romanticToastRepo := repository.NewRomanticToastRepo(pool)
	romanticToastHandler := handlers.NewRomanticToastHandler(romanticToastRepo)
	gameHandler := handlers.NewGameHandler(gameRepo)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	// 静态访问已落盘图片：GET /uploads/travel-diaries/1/1.jpg
	r.Static(storage.URLPrefix, fileStore.Root())
	// 单文件上传上限 10MB（与 upload handler 一致）
	r.MaxMultipartMemory = 10 << 20

	// Docker 健康检查：探测数据库连通性
	r.GET("/romantic-toasts/random", romanticToastHandler.Random)

	r.GET("/health", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	handlers.RegisterAuthRoutes(r, authHandler, cfg)

	r.POST("/upload/image", uploadHandler.UploadImage)
	r.POST("/upload/images", uploadHandler.UploadImages)
	r.POST("/upload/avatar", middleware.Auth(cfg), uploadHandler.UploadAvatar)

	registerDiaryRoutes(r, "/travel-diaries", travelHandler)
	registerDiaryRoutes(r, "/daily-diaries", dailyHandler)
	handlers.RegisterCommentRoutes(r, "/travel-diaries", travelCommentHandler, cfg)
	handlers.RegisterCommentRoutes(r, "/daily-diaries", dailyCommentHandler, cfg)

	r.GET("/whispers", whisperHandler.List)
	r.GET("/whispers/:id", whisperHandler.GetOne)
	r.POST("/whispers", whisperHandler.Create)
	r.PUT("/whispers/:id", whisperHandler.Update)
	r.PUT("/whispers/:id/like", whisperHandler.Like)
	r.DELETE("/whispers/:id", whisperHandler.Delete)

	r.GET("/special-dates", specialDateHandler.List)
	r.GET("/special-dates/:id", specialDateHandler.GetOne)
	r.POST("/special-dates", specialDateHandler.Create)
	r.PUT("/special-dates/:id", specialDateHandler.Update)
	r.DELETE("/special-dates/:id", specialDateHandler.Delete)

	handlers.RegisterGameRoutes(r, gameHandler)

	r.GET("/tree", treeHandler.Get)
	r.GET("/tree/logs", treeHandler.Logs)
	// 浇水需登录（与原 Node 服务一致）
	r.POST("/tree/water", middleware.Auth(cfg), treeHandler.Water)

	addr := ":" + cfg.Port
	log.Printf("love-diary-go listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

// registerDiaryRoutes 注册旅行/日常日记的通用 CRUD 路由。
func registerDiaryRoutes(r *gin.Engine, base string, h *handlers.DiaryHandler) {
	r.GET(base, h.List)
	r.GET(base+"/:id", h.GetOne)
	r.POST(base, h.Create)
	r.PUT(base+"/:id", h.Update)
	r.PUT(base+"/:id/like", h.Like)
	r.DELETE(base+"/:id", h.Delete)
}
