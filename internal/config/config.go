package config

import "github.com/spf13/viper"

type ServerConfig struct {
	Port                string
	DatabaseDSN         string
	RedisURL            string
	MessageHistoryLimit int
	RoomListLimit       int
	RateLimitRequests   int
	RateLimitWindowSecs int
}

func LoadServerConfig() ServerConfig {
	viper.SetDefault("server.port", ":8080")
	viper.SetDefault("server.database_dsn", "postgres://root:password@localhost:5432/chatatui?sslmode=disable")
	viper.SetDefault("server.redis_url", "redis://localhost:6379")
	viper.SetDefault("server.message_history_limit", 50)
	viper.SetDefault("server.room_list_limit", 100)
	viper.SetDefault("server.rate_limit_requests", 100)
	viper.SetDefault("server.rate_limit_window_secs", 60)

	return ServerConfig{
		Port:                viper.GetString("server.port"),
		DatabaseDSN:         viper.GetString("server.database_dsn"),
		RedisURL:            viper.GetString("server.redis_url"),
		MessageHistoryLimit: viper.GetInt("server.message_history_limit"),
		RoomListLimit:       viper.GetInt("server.room_list_limit"),
		RateLimitRequests:   viper.GetInt("server.rate_limit_requests"),
		RateLimitWindowSecs: viper.GetInt("server.rate_limit_window_secs"),
	}
}
