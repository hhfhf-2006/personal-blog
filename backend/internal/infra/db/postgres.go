package db

import (
	"fmt"
	"net/url"
	"time"

	"personal-blog-backend/internal/config"
	"personal-blog-backend/internal/pkg/timeutil"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(cfg config.PostgresConfig) (*gorm.DB, error) {
	// 使用 URL 格式 DSN，由 net/url 自动处理特殊字符编码，避免 fmt.Sprintf 格式化漏洞和空格/引号等特殊字符问题
	// TimeZone=Asia/Shanghai：将数据库会话时区固定为东八区，使服务端默认值
	// （NOW()/CURRENT_TIMESTAMP）及任何时区相关运算都以北京时间为准。
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=Asia/Shanghai",
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		url.QueryEscape(cfg.Host),
		url.QueryEscape(cfg.Port),
		url.QueryEscape(cfg.DBName),
		url.QueryEscape(cfg.SSLMode),
	)

	// NowFunc 让 GORM 自动维护的 CreatedAt/UpdatedAt 以及所有 time.Now 语义
	// 统一使用北京时间，确保写入 `timestamp` 列的墙上时钟始终是东八区。
	//
	// TranslateError 将底层驱动的原生错误（如 PostgreSQL 唯一键冲突 23505）翻译为
	// GORM 的语义化错误（gorm.ErrDuplicatedKey 等）。点赞切换、用户注册、游戏分数等
	// 多处依赖 errors.Is(err, gorm.ErrDuplicatedKey) 判断冲突，不开启此项会导致这些
	// 判断恒为 false —— 是「取消点赞失效」等问题的根因之一，必须开启。
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc:        timeutil.Now,
		TranslateError: true,
	})
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

	// 配置连接池，避免死连接导致请求卡死
	// 生产环境建议根据负载调整，对于个人博客这些值是合理的
	sqlDB.SetMaxOpenConns(25)                  // 最大打开连接数
	sqlDB.SetMaxIdleConns(5)                   // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(3 * time.Minute)  // 连接最大存活 3 分钟，到期自动回收
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)  // 空闲连接超过 1 分钟自动关闭

	return database, nil
}