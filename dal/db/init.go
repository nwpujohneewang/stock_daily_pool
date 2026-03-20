package db

import (
	"fmt"
	"stock/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init initializes the PostgreSQL connection via GORM.
func Init() {
	dsn := config.GlobalConfig.Database.DSN()

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
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

	fmt.Println("PostgreSQL connected successfully via GORM")
}
