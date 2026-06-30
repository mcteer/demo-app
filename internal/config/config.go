package config

import (
	"fmt"
	"os"
)

type DBConfig struct {
	Host    string
	Port    string
	Name    string
	User    string
	Pass    string
	SSLMode string
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

	db := DBConfig{
		Host:    os.Getenv("DB_HOST"),
		Port:    os.Getenv("DB_PORT"),
		Name:    os.Getenv("DB_NAME"),
		User:    os.Getenv("DB_USER"),
		SSLMode: os.Getenv("DB_SSLMODE"),
	}
	db.Pass = os.Getenv("DB_PASSWORD")

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
	if db.Pass == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD is required")
	}
	if db.SSLMode == "" {
		db.SSLMode = "disable"
	}

	return Config{
		Port: port,
		DB:   db,
	}, nil
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Pass, d.SSLMode,
	)
}
