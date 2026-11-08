// Package cache encapsula el cliente Redis y operaciones tolerantes a fallos.
// Si Redis no está disponible, todas las operaciones se convierten en no-ops
// para que la API siga funcionando contra SQLite.
package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/xavie/gastos-api/internal/config"
)

const operationTimeout = 500 * time.Millisecond

// Client es un cliente Redis con degradación elegante.
type Client struct {
	rdb  *redis.Client
	log  *slog.Logger
	addr string
}

// New crea el cliente. Si REDIS_ENABLED=false devuelve un cliente inerte.
// Si Redis no responde en el arranque, registra un warning y mantiene el
// cliente creado para que la API se recupere cuando Redis vuelva.
func New(cfg *config.Config, log *slog.Logger) *Client {
	if !cfg.RedisEnabled {
		log.Info("Redis deshabilitado; la API funcionará sin caché ni rate limiting distribuido")
		return &Client{log: log}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DialTimeout:  operationTimeout,
		ReadTimeout:  operationTimeout,
		WriteTimeout: operationTimeout,
		MaxRetries:   1,
	})

	client := &Client{rdb: rdb, log: log, addr: cfg.RedisAddr}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("Redis no disponible en el arranque; se continuará sin caché hasta que vuelva",
			"addr", cfg.RedisAddr,
			"error", err,
		)
		return client
	}

	log.Info("conexión con Redis establecida", "addr", cfg.RedisAddr)
	return client
}

// Enabled indica si hay un backend Redis configurado.
func (c *Client) Enabled() bool {
	return c != nil && c.rdb != nil
}

// Redis expone el cliente subyacente para el middleware de rate limiting.
func (c *Client) Redis() *redis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// Ping comprueba la disponibilidad de Redis.
func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	return c.rdb.Ping(ctx).Err()
}

// Close cierra la conexión con Redis.
func (c *Client) Close() error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Close()
}

// GetJSON recupera y deserializa un valor. Devuelve false si la clave no existe
// o si Redis falla; en el segundo caso deja un warning en el log.
func (c *Client) GetJSON(ctx context.Context, key string, destination any) bool {
	if !c.Enabled() {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			c.warn("leyendo caché", key, err)
		}
		return false
	}

	if err := json.Unmarshal(raw, destination); err != nil {
		c.warn("deserializando caché", key, err)
		_ = c.rdb.Del(ctx, key).Err()
		return false
	}
	return true
}

// SetJSON serializa y guarda un valor con TTL.
func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	if !c.Enabled() {
		return
	}

	raw, err := json.Marshal(value)
	if err != nil {
		c.warn("serializando caché", key, err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := c.rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		c.warn("guardando caché", key, err)
	}
}

// Delete elimina una o varias claves.
func (c *Client) Delete(ctx context.Context, keys ...string) {
	if !c.Enabled() || len(keys) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		c.warn("eliminando caché", keys[0], err)
	}
}

// DeleteByPattern elimina todas las claves que cumplen un patrón usando SCAN.
func (c *Client) DeleteByPattern(ctx context.Context, pattern string) {
	if !c.Enabled() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.warn("buscando claves de caché", pattern, err)
			return
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				c.warn("eliminando claves de caché", pattern, err)
				return
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func (c *Client) warn(operation, key string, err error) {
	if c.log == nil {
		return
	}
	c.log.Warn("fallo de Redis; se continúa sin caché",
		"operation", operation,
		"key", key,
		"addr", c.addr,
		"error", err,
	)
}
