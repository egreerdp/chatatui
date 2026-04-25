package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Addr                string           `mapstructure:"addr"`
	DatabaseDSN         string           `mapstructure:"database_dsn"`
	RedisURL            string           `mapstructure:"redis_url"`
	MessageHistoryLimit int              `mapstructure:"message_history_limit"`
	RoomListLimit       int              `mapstructure:"room_list_limit"`
	RateLimitRequests   int              `mapstructure:"rate_limit_requests"`
	RateLimitWindowSecs int              `mapstructure:"rate_limit_window_secs"`
	Federation          FederationConfig `mapstructure:"federation"`
}

type FederationConfig struct {
	Domain  string `mapstructure:"domain"`
	Enabled bool   `mapstructure:"enabled"`
}

func (c ServerConfig) Validate() error {
	if c.Federation.Enabled && c.Federation.Domain == "" {
		return errors.New("server.federation.domain is required when federation is enabled; set it or disable federation with server.federation.enabled = false")
	}
	return nil
}

func LoadServerConfig() (ServerConfig, error) {
	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("server.database_dsn", "postgres://root:password@localhost:5432/chatatui?sslmode=disable")
	viper.SetDefault("server.redis_url", "redis://localhost:6379")
	viper.SetDefault("server.message_history_limit", 50)
	viper.SetDefault("server.room_list_limit", 100)
	viper.SetDefault("server.rate_limit_requests", 100)
	viper.SetDefault("server.rate_limit_window_secs", 60)
	viper.SetDefault("server.federation.enabled", true)

	var cfg ServerConfig
	if err := viper.UnmarshalKey("server", &cfg); err != nil {
		return ServerConfig{}, fmt.Errorf("parsing server config: %w", err)
	}

	return cfg, nil
}
