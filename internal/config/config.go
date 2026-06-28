package config

import (
	"fmt"
	"os"
)

type DBConfig struct {
	DSN string
}

type Config struct {
	Port string
	DB   DBConfig
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return Config{
			Port: port,
			DB:   DBConfig{DSN: dsn},
		}, nil
	}

	host := os.Getenv("DB_HOST")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	dbPort := os.Getenv("DB_PORT")
	sslMode := os.Getenv("DB_SSLMODE")
	credential := os.Getenv("DB_PASSWORD")

	if host == "" {
		return Config{}, fmt.Errorf("DB_HOST is required")
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if name == "" {
		return Config{}, fmt.Errorf("DB_NAME is required")
	}
	if user == "" {
		return Config{}, fmt.Errorf("DB_USER is required")
	}
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s sslmode=%s",
		host, dbPort, name, user, sslMode,
	)
	if credential != "" {
		dsn = fmt.Sprintf("%s password=%s", dsn, credential)
	}

	return Config{
		Port: port,
		DB:   DBConfig{DSN: dsn},
	}, nil
}

func (d DBConfig) ConnectionString() string {
	return d.DSN
}
