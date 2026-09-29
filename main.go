package main

import (
	"context"
	"fmt"
	"ginblog/config"
	"ginblog/middleware"
	"ginblog/model"
	"ginblog/pkg/database"
	"ginblog/pkg/logger"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	closer, err := logger.Init(logger.Config(cfg.Logger))
	if err != nil {
		slog.Error("failed to init logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer closer.Close()
	slog.Info("init logger success")

	// 初始化数据库（GORM 日志桥接到 slog）
	gormLogger := logger.NewGormLogger(cfg.Logger.Level, 200*time.Millisecond)
	db, err := database.Init(database.Config{
		File:   cfg.Database.File,
		Logger: gormLogger,
	})
	if err != nil {
		slog.Error("failed to init database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 迁移所有表结构
	migrateErr := database.Migrate(db, []any{
		&model.Article{},
	})
	if migrateErr != nil {
		slog.Error("failed to migrate database", slog.String("error", migrateErr.Error()))
		os.Exit(1)
	}

	// 初始化Gin服务和路由
	r := gin.New()

	r.Use(middleware.Logger())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "hello world",
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", slog.String("error", err.Error()))
		}
	}()

	slog.Info("start server success")

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(ctx)
	if err != nil {
		slog.Error("shutdown error", slog.String("error", err.Error()))
	} else {
		slog.Info("shutdown server success")
	}
}
