package repository

import (
	"fmt"
	"log"
	"time"

	"funchat/backend/config"
	"funchat/backend/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB 创建数据库连接并自动迁移
func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConn)
	sqlDB.SetMaxIdleConns(cfg.MaxConn / 4)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// 自动迁移（AutoMigrate）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Friendship{},
		&model.FriendRequest{},
		&model.Message{},
		&model.ConditionConfig{},
		&model.UnlockRecord{},
		&model.BurnRecord{},
		&model.QuizQuestion{},
		&model.VerificationCode{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("[DB] 数据库连接成功 + AutoMigrate 完成 (9 张表)")
	return db, nil
}
