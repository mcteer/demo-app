package config

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConfig struct {
	Host       string
	Port       string
	Name       string
	User       string
	Credential string
	SSLMode    string
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
		Host:       os.Getenv("DB_HOST"),
		Port:       os.Getenv("DB_PORT"),
		Name:       os.Getenv("DB_NAME"),
		User:       os.Getenv("DB_USER"),
		Credential: os.Getenv("DB_PASSWORD"),
		SSLMode:    os.Getenv("DB_SSLMODE"),
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
	if db.Credential == "" {
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

func (d DBConfig) PoolConfig() (*pgxpool.Config, error) {
	connURL := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(d.User, d.Credential),
		Host:   net.JoinHostPort(d.Host, d.Port),
		Path:   "/" + d.Name,
	}
	query := connURL.Query()
	query.Set("sslmode", d.SSLMode)
	connURL.RawQuery = query.Encode()

	return pgxpool.ParseConfig(connURL.String())
}
