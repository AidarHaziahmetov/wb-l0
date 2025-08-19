package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppName       string   `envconfig:"APP_NAME" default:"wb-l0-go"`
	HTTPAddr      string   `envconfig:"HTTP_ADDR" default:":8080"`
	KafkaBrokers  []string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	KafkaTopic    string   `envconfig:"KAFKA_TOPIC" default:"orders"`
	KafkaGroupID  string   `envconfig:"KAFKA_GROUP_ID" default:"wb-l0-go-consumer"`
	PostgresDSN   string   `envconfig:"PG_DSN" default:"postgres://postgres:postgres@localhost:5432/l0?sslmode=disable"`
	LogLevel      string   `envconfig:"LOG_LEVEL" default:"info"`
	CacheMaxItems int      `envconfig:"CACHE_MAX_ITEMS" default:"100"`
}

// Загрузка конфигурации из переменных окружения и файла .env.
func Load() (Config, error) {
	var cfg Config
	_ = godotenv.Load()
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
