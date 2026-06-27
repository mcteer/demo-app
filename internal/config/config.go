package config

import (
	"fmt"
	"os"
	"strings"
)

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
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

	user, err := loadCredential("DB_USER", "DB_USER_FILE")
	if err != nil {
		return Config{}, err
	}
	pass, err := loadCredential("DB_PASSWORD", "DB_PASSWORD_FILE")
	if err != nil {
		return Config{}, err
	}

	db := DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
		User:     user,
		Password: pass,
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	if db.Host == "" {
		return Config{}, fmt.Errorf("DB_HOST is required")
	}
	if db.Port == "" {
		db.Port = "5432"
	}
	if db.Name == "" {
		return Config{}, fmt.Errorf("DB_NAME is required")
	}
	if db.User == "" {
		return Config{}, fmt.Errorf("DB_USER is required")
	}
	if db.SSLMode == "" {
		db.SSLMode = "disable"
	}

	return Config{
		Port: port,
		DB:   db,
	}, nil
}

func loadCredential(envKey, fileEnvKey string) (string, error) {
	if path := os.Getenv(fileEnvKey); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fileEnvKey, err)
		}
		if v := strings.TrimSpace(string(data)); v != "" {
			return v, nil
		}
	}
	return os.Getenv(envKey), nil
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}
