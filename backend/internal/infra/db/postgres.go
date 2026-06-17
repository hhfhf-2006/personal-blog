package db

import (
	"fmt"

	"personal-blog-backend/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(cfg config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 获取底层的 *sql.DB，然后真正 Ping 一下数据库
	// gorm.Open 只会校验 DSN 格式，不会验证连接是否真的能通
	// Ping() 才会真正发一次网络请求，确认数据库在不在、密码对不对
	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败，请检查 PostgreSQL 是否已启动: %w", err)
	}

	return database, nil
}