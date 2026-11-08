// Package database gestiona la conexión con SQLite y las migraciones.
package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xavie/gastos-api/internal/config"
	"github.com/xavie/gastos-api/internal/models"
)

// Connect abre la base de datos SQLite, aplica los pragmas recomendados y
// ejecuta las migraciones automáticas de GORM.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn, err := buildDSN(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("abriendo base de datos SQLite: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("obteniendo conexión subyacente: %w", err)
	}

	// SQLite serializa las escrituras; una sola conexión evita bloqueos y
	// mantiene coherente el modo :memory: usado en los tests.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("verificando conexión a SQLite: %w", err)
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("activando claves foráneas: %w", err)
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, fmt.Errorf("configurando busy_timeout: %w", err)
	}
	if dsn != ":memory:" {
		// WAL mejora la concurrencia de lectura en archivos; en memoria no aplica.
		_ = db.Exec("PRAGMA journal_mode = WAL").Error
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

// Migrate crea o actualiza el esquema de todas las entidades.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Transaction{},
		&models.Budget{},
		&models.RefreshToken{},
	); err != nil {
		return fmt.Errorf("ejecutando migraciones: %w", err)
	}
	return nil
}

func buildDSN(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("DB_PATH no puede estar vacío")
	}
	if path == ":memory:" || strings.HasPrefix(path, "file:") {
		return path, nil
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("creando directorio de base de datos %q: %w", dir, err)
		}
	}

	return "file:" + filepath.ToSlash(path), nil
}
