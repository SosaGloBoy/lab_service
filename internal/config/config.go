package config

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"lab/internal/model"
	"os"
)

type Config struct {
	DBUser         string
	DBPassword     string
	DBName         string
	DBHost         string
	DBPort         string
	JWTSecret      string
	TaskServiceURL string
	ServerPort     string
}

func LoadConfig() Config {
	return Config{
		DBUser:         getEnv("DB_USER", "user"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "lab_db"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5434"),
		JWTSecret:      getEnv("JWT_SECRET", "my_secret_key"),
		TaskServiceURL: getEnv("TASK_SERVICE_URL", "http://localhost:8086"),
		ServerPort:     getEnv("SERVER_PORT", ":8082"),
	}
}

func InitDB(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.AutoMigrate(&model.Lab{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
