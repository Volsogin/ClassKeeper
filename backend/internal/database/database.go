package database

import (
	"fmt"
	"log"

	"classkeeper/internal/config"
	"classkeeper/internal/models"

	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect устанавливает соединение с базой данных
func Connect(cfg *config.Config) error {
	var dialector gorm.Dialector

	switch cfg.Database.Type {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			cfg.Database.Host,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
			cfg.Database.Port,
		)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(cfg.Database.SQLPath)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	// Настройка логирования
	logLevel := logger.Silent
	if cfg.Server.Environment == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	log.Printf("Connected to %s database successfully", cfg.Database.Type)
	return nil
}

// Migrate выполняет автоматическую миграцию схемы базы данных
func Migrate() error {
	log.Println("Running database migrations...")

	err := DB.AutoMigrate(
		&models.School{},
		&models.User{},
		&models.Class{},
		&models.Subject{},
		&models.Schedule{},
		&models.Attendance{},
		&models.Grade{},
		&models.Homework{},
		&models.Announcement{},
		&models.ParentStudent{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// Close закрывает соединение с базой данных
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
