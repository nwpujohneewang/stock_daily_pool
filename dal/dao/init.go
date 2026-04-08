package dao

import (
	"context"
	"fmt"
	"stock/config"
	"time"

	pkglogger "stock/internal/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

var shanghaiLoc *time.Location

func init() {
	var err error
	shanghaiLoc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(fmt.Errorf("failed to load Asia/Shanghai timezone: %w", err))
	}
}

func Now() time.Time {
	return time.Now().In(shanghaiLoc)
}

func Init() {
	dsn := config.GlobalConfig.Database.DSN()

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		panic(fmt.Errorf("failed to open gorm db: %w", err))
	}

	sqlDB, err := DB.DB()
	if err != nil {
		panic(fmt.Errorf("failed to get underlying sql.DB: %w", err))
	}

	sqlDB.SetMaxOpenConns(config.GlobalConfig.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.GlobalConfig.Database.MaxIdleConns)
	if config.GlobalConfig.Database.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(config.GlobalConfig.Database.ConnMaxLifetime) * time.Second)
	}
	if config.GlobalConfig.Database.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(config.GlobalConfig.Database.ConnMaxIdleTime) * time.Second)
	}

	pkglogger.Info("PostgreSQL connected successfully via GORM",
		zap.Int("max_open_conns", config.GlobalConfig.Database.MaxOpenConns),
		zap.Int("max_idle_conns", config.GlobalConfig.Database.MaxIdleConns),
	)
}

func PostgresStockDB(ctx context.Context) *gorm.DB {
	return DB.WithContext(ctx)
}
