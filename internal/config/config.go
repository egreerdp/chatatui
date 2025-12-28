package config

import "github.com/spf13/viper"

type ServerConfig struct {
	Port                string
	Database            string
	MessageHistoryLimit int
	RoomListLimit       int
}

func LoadServerConfig() ServerConfig {
	viper.SetDefault("server.port", ":8080")
	viper.SetDefault("server.database", "chatatui.db")
	viper.SetDefault("server.message_history_limit", 50)
	viper.SetDefault("server.room_list_limit", 100)

	return ServerConfig{
		Port:                viper.GetString("server.port"),
		Database:            viper.GetString("server.database"),
		MessageHistoryLimit: viper.GetInt("server.message_history_limit"),
		RoomListLimit:       viper.GetInt("server.room_list_limit"),
	}
}
