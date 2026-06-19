package config

import (
	"errors"
	"os"
)

type Config struct {
	ServerPort    string
	JWTSecret     string
	AdminEmail    string
	AdminUsername string
	AdminPassword string
	Postgres      PostgresConfig
	GitHub        GitHubOAuthConfig
}

type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
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
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		AdminEmail:    getEnv("ADMIN_EMAIL", "183976823@qq.com"),
		AdminUsername: getEnv("ADMIN_USERNAME", "灰化肥发灰"),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		Postgres: PostgresConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "blog"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "blog_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		GitHub: GitHubOAuthConfig{
			ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
			ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			RedirectURI:  getEnv("GITHUB_REDIRECT_URI", ""),
		},
	}
}

// Validate 检查必须的配置项是否已设置
func (c Config) Validate() error {
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET 未设置 —— 请在 .env 文件中设置一个随机字符串作为 JWT 签名密钥")
	}
	return nil
}

// getEnv 从环境变量读取值，如果不存在则返回默认值
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}