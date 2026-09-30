package main

import (
	"context"
	"fmt"
	"ginblog/api"
	"ginblog/config"
	"ginblog/middleware"
	"ginblog/model"
	"ginblog/pkg/database"
	"ginblog/pkg/logger"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 读取配置
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 初始化日志
	logs, err := logger.Init(logger.Config(cfg.Logger))
	if err != nil {
		slog.Error("failed to init logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer logs.Close()
	slog.Info("init logger success")

	// 日志初始化之后的失败退出：先关闭日志文件（os.Exit 不会执行 defer）
	fatalf := func(msg string, err error) {
		slog.Error(msg, slog.String("error", err.Error()))
		_ = logs.Close()
		os.Exit(1)
	}

	// 初始化数据库（GORM 日志桥接到 slog，不带 source）
	gormLogger := logger.NewGormLogger(logs.Gorm, cfg.Logger.Level, 200*time.Millisecond)
	db, err := database.Init(database.Config{
		File:   cfg.Database.File,
		Logger: gormLogger,
	})
	if err != nil {
		fatalf("failed to init database", err)
	} else {
		slog.Info("init database success")
	}

	defer func() {
		sqlDB, _ := db.DB()
		err := sqlDB.Close()
		if err != nil {
			slog.Error("failed to close sqlDB", slog.String("error", err.Error()))
		} else {
			slog.Info("close sqlDB")
		}
	}()

	// 迁移所有表结构
	if migrateErr := database.Migrate(db, []any{
		&model.Article{},
	}); migrateErr != nil {
		fatalf("failed to migrate database", migrateErr)
	} else {
		slog.Info("migrate all database success")
	}

	// 初始化Gin服务和路由
	// gin 运行模式：默认 release，调试时设置 GIN_MODE=debug
	switch strings.ToLower(strings.TrimSpace(cfg.Server.ServerMode)) {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	// gin 自身日志转发到 slog，避免与 JSON 格式混排
	gin.DefaultWriter = logger.NewSlogWriter("gin", slog.LevelInfo)
	gin.DefaultErrorWriter = logger.NewSlogWriter("gin", slog.LevelError)

	r := gin.New()

	// Logger 在外层、Recovery 在内层：panic 恢复后仍能记录访问日志
	r.Use(middleware.Logger(), middleware.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "hello world",
		})
	})

	api.Init(db, r)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// 监听失败（如端口占用）通过 channel 通知主 goroutine，避免进程假死：
	// 只在真正的错误时发送（Shutdown 导致的 ErrServerClosed 不发），
	// buffered=1 保证即使无人接收 goroutine 也不会阻塞
	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	slog.Info("start server success")

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		// 启动即失败：记录日志、关闭日志文件并退出（进程不能无监听地活着）
		fatalf("failed to listen and serve", err)
	case sig := <-quit:
		slog.Info("received signal, shutting down", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", slog.String("error", err.Error()))
		} else {
			slog.Info("shutdown server success")
		}
	}
}
