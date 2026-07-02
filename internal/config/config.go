package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig
	JWT         JWTConfig
	Internal    InternalConfig
	Upstreams   UpstreamConfig
	RateLimit   RateLimitConfig
	CORS        CORSConfig
}

type ServerConfig struct {
	Port string
}

type JWTConfig struct {
	Secret string
}

// InternalConfig guards api-gateway's /internal/* endpoints — reachable only from
// other services on snaply-net, never from client traffic, so it's checked via a
// shared secret header instead of a JWT.
type InternalConfig struct {
	Secret string
}

type UpstreamConfig struct {
	UserServiceURL         string
	PostServiceURL         string
	RelationServiceURL     string
	LikeServiceURL         string
	NotificationServiceURL string
	MessageServiceURL      string
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("USER_SERVICE_URL", "http://localhost:8081")
	viper.SetDefault("POST_SERVICE_URL", "http://localhost:8082")
	viper.SetDefault("RELATION_SERVICE_URL", "http://localhost:8083")
	viper.SetDefault("LIKE_SERVICE_URL", "http://localhost:8084")
	viper.SetDefault("NOTIFICATION_SERVICE_URL", "http://localhost:8085")
	viper.SetDefault("MESSAGE_SERVICE_URL", "http://localhost:8087")
	viper.SetDefault("RATE_LIMIT_RPS", 100.0)
	viper.SetDefault("RATE_LIMIT_BURST", 200)
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")

	viper.AutomaticEnv()

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetString("SERVER_PORT"),
		},
		JWT: JWTConfig{
			Secret: viper.GetString("JWT_SECRET"),
		},
		Internal: InternalConfig{
			Secret: viper.GetString("INTERNAL_SECRET"),
		},
		Upstreams: UpstreamConfig{
			UserServiceURL:         viper.GetString("USER_SERVICE_URL"),
			PostServiceURL:         viper.GetString("POST_SERVICE_URL"),
			RelationServiceURL:     viper.GetString("RELATION_SERVICE_URL"),
			LikeServiceURL:         viper.GetString("LIKE_SERVICE_URL"),
			NotificationServiceURL: viper.GetString("NOTIFICATION_SERVICE_URL"),
			MessageServiceURL:      viper.GetString("MESSAGE_SERVICE_URL"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: viper.GetFloat64("RATE_LIMIT_RPS"),
			Burst:             viper.GetInt("RATE_LIMIT_BURST"),
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(viper.GetString("CORS_ALLOWED_ORIGINS"), ","),
		},
	}

	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = "dev_secret_change_in_production"
	}
	if cfg.Internal.Secret == "" {
		cfg.Internal.Secret = "dev_internal_secret_change_in_production"
	}

	return cfg, nil
}
