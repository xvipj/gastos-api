// Package config centraliza la carga y validación de la configuración de la API.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config contiene toda la configuración necesaria para arrancar la aplicación.
type Config struct {
	Port            string
	GinMode         string
	DBPath          string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	RedisEnabled    bool
	RedisAddr       string
	RedisPassword   string
}

// Load lee el archivo .env (si existe) y las variables de entorno del sistema.
// Las variables de entorno siempre tienen prioridad sobre el contenido del .env.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("cargando archivo .env: %w", err)
	}

	accessTTL, err := durationFromEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshTTL, err := durationFromEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:            envOrDefault("PORT", "8080"),
		GinMode:         envOrDefault("GIN_MODE", "debug"),
		DBPath:          envOrDefault("DB_PATH", "gastos.db"),
		JWTSecret:       strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		RedisEnabled:    boolFromEnv("REDIS_ENABLED", false),
		RedisAddr:       envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate comprueba que la configuración sea coherente antes de arrancar.
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET es obligatorio")
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET debe tener al menos 16 caracteres")
	}
	if c.Port == "" {
		return fmt.Errorf("PORT no puede estar vacío")
	}
	if c.DBPath == "" {
		return fmt.Errorf("DB_PATH no puede estar vacío")
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL debe ser mayor que cero")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL debe ser mayor que cero")
	}
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		return fmt.Errorf("REFRESH_TOKEN_TTL debe ser mayor que ACCESS_TOKEN_TTL")
	}
	if c.RedisEnabled && c.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR es obligatorio cuando REDIS_ENABLED=true")
	}
	switch c.GinMode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("GIN_MODE debe ser debug, release o test")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("valor inválido para %s (%q): %w", key, raw, err)
	}
	return value, nil
}

func boolFromEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
