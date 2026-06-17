package config

import "os"

type Config struct {
	ServerPort string
	JWTSecret  string
	Postgres   PostgresConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Load 读取配置。优先使用环境变量，环境变量不存在时使用默认值。
func Load() Config {
	return Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		Postgres: PostgresConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "blog"),
			Password: getEnv("DB_PASSWORD", "blog123456"),
			DBName:   getEnv("DB_NAME", "blog_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
}

// getEnv 从环境变量读取值，如果不存在则返回默认值
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}