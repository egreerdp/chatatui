package config

import "github.com/spf13/viper"

type ServerConfig struct {
	Port                string
	DatabaseDSN         string
	MessageHistoryLimit int
	RoomListLimit       int
}

func LoadServerConfig() ServerConfig {
	viper.SetDefault("server.port", ":8080")
	viper.SetDefault("server.database_dsn", "postgres://root:password@localhost:5432/chatatui?sslmode=disable")
	viper.SetDefault("server.message_history_limit", 50)
	viper.SetDefault("server.room_list_limit", 100)

	return ServerConfig{
		Port:                viper.GetString("server.port"),
		DatabaseDSN:         viper.GetString("server.database_dsn"),
		MessageHistoryLimit: viper.GetInt("server.message_history_limit"),
		RoomListLimit:       viper.GetInt("server.room_list_limit"),
	}
}
