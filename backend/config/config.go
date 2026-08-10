package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	SMS      SMSConfig
	COS      COSConfig
}

// ServerConfig 服务配置
type ServerConfig struct {
	Environment  string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	AllowedOrigins []string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	MaxConn  int
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret           string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
}

// SMSConfig 短信配置
type SMSConfig struct {
	SecretID  string
	SecretKey string
	AppID     string
	SignName  string
	TemplateID string
}

// COSConfig COS 配置
type COSConfig struct {
	SecretID  string
	SecretKey string
	Bucket    string
	Region    string
}

// Load 加载配置
func Load() (*Config, error) {
	// 加载 .env 文件（开发环境）
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Environment:  getEnv("APP_ENV", "development"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			AllowedOrigins: splitCSV(getEnv("ALLOWED_ORIGINS", "")),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "funchat"),
			MaxConn:  100,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "funchat-dev-secret-change-in-production"),
			AccessTokenTTL:  2 * time.Hour,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
		SMS: SMSConfig{
			SecretID:   getEnv("SMS_SECRET_ID", ""),
			SecretKey:  getEnv("SMS_SECRET_KEY", ""),
			AppID:      getEnv("SMS_APP_ID", ""),
			SignName:   getEnv("SMS_SIGN_NAME", ""),
			TemplateID: getEnv("SMS_TEMPLATE_ID", ""),
		},
		COS: COSConfig{
			SecretID:  getEnv("COS_SECRET_ID", ""),
			SecretKey: getEnv("COS_SECRET_KEY", ""),
			Bucket:    getEnv("COS_BUCKET", ""),
			Region:    getEnv("COS_REGION", ""),
		},
	}, validate()
}

func validate() error {
	if getEnv("APP_ENV", "development") != "production" {
		return nil
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 || secret == "funchat-dev-secret-change-in-production" || secret == "funchat-prod-secret" {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	if len(splitCSV(os.Getenv("ALLOWED_ORIGINS"))) == 0 {
		return fmt.Errorf("ALLOWED_ORIGINS is required in production")
	}
	return nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
