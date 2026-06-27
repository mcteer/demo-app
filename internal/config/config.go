package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
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
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Credential: os.Getenv("DB_PASSWORD"),
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

func (d DBConfig) ConnectionURL() (string, error) {
	if d.Host == "" || d.Name == "" || d.User == "" || d.Credential == "" {
		return "", fmt.Errorf("incomplete database configuration")
	}

	port := d.Port
	if port == "" {
		port = "5432"
	}

	sslmode := d.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(d.Host, port),
		Path:   "/" + d.Name,
	}
	u.User = url.UserPassword(d.User, d.Credential)

	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
