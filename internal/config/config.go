package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config stores all runtime configuration for the bot.
type Config struct {
	Environment string
	LogLevel    string

	Discord DiscordConfig
	Mongo   MongoConfig
	Casino  CasinoConfig
}

// DiscordConfig contains Discord API settings.
type DiscordConfig struct {
	Token         string
	GuildID       string
	ProtectedUser string
	OwnerName     string
}

// MongoConfig contains MongoDB settings.
type MongoConfig struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
}

// CasinoConfig contains gameplay settings.
type CasinoConfig struct {
	DefaultBalance int64
	RandomSeed     int64
}

// Load reads and validates configuration from environment variables and optional .env.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("DISCORD_GUILD_ID", "")
	v.SetDefault("PROTECTED_USER", "hellomimiz")
	v.SetDefault("DON_RAT_OWNER_NAME", "Don Rat")
	v.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	v.SetDefault("MONGO_DATABASE", "donrat")
	v.SetDefault("MONGO_CONNECT_TIMEOUT_SECONDS", 10)
	v.SetDefault("CASINO_DEFAULT_BALANCE", 1000)
	v.SetDefault("CASINO_RANDOM_SEED", 42)

	cfg := Config{
		Environment: v.GetString("ENVIRONMENT"),
		LogLevel:    v.GetString("LOG_LEVEL"),
		Discord: DiscordConfig{
			Token:         v.GetString("DISCORD_TOKEN"),
			GuildID:       v.GetString("DISCORD_GUILD_ID"),
			ProtectedUser: v.GetString("PROTECTED_USER"),
			OwnerName:     v.GetString("DON_RAT_OWNER_NAME"),
		},
		Mongo: MongoConfig{
			URI:            v.GetString("MONGO_URI"),
			Database:       v.GetString("MONGO_DATABASE"),
			ConnectTimeout: time.Duration(v.GetInt("MONGO_CONNECT_TIMEOUT_SECONDS")) * time.Second,
		},
		Casino: CasinoConfig{
			DefaultBalance: v.GetInt64("CASINO_DEFAULT_BALANCE"),
			RandomSeed:     v.GetInt64("CASINO_RANDOM_SEED"),
		},
	}

	if cfg.Discord.Token == "" {
		return Config{}, fmt.Errorf("DISCORD_TOKEN is required")
	}
	if cfg.Mongo.URI == "" {
		return Config{}, fmt.Errorf("MONGO_URI is required")
	}
	if cfg.Mongo.Database == "" {
		return Config{}, fmt.Errorf("MONGO_DATABASE is required")
	}
	if cfg.Casino.DefaultBalance <= 0 {
		return Config{}, fmt.Errorf("CASINO_DEFAULT_BALANCE must be greater than zero")
	}

	return cfg, nil
}
