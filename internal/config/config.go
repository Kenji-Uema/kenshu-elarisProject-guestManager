package config

import (
	"context"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Secret string

func (s Secret) String() string {
	return "REDACTED"
}

type Configs struct {
	AppConfig
	Services
	MongoConfig
	RabbitMqConfig
	RedisConfig
	TelemetryConfig
}

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME" envDefault:"kenshu-elarisProject-guestManager"`
	Version     string `env:"VERSION"`
	Server      struct {
		Host string `env:"SERVICE_HOST,required"`
		Port int    `env:"SERVICE_PORT,required"`
	}
}

type Services struct {
	ClockSimulator struct {
		GrpcHost string `env:"CLOCK_EMU_GRPC_HOST,required"`
		GrpcPort int    `env:"CLOCK_EMU_GRPC_PORT,required"`
	}
}

type MongoConfig struct {
	Username    Secret `env:"MONGO_INITDB_ROOT_USERNAME,required"`
	Password    Secret `env:"MONGO_INITDB_ROOT_PASSWORD,required"`
	Host        string `env:"MONGO_HOST,required"`
	Database    string `env:"MONGO_DATABASE,required"`
	Collections struct {
		Guest   string `env:"GUEST_COLLECTION,required" envDefault:"Guest"`
		Cottage string `env:"COTTAGE_COLLECTION,required" envDefault:"Cottage"`
		Booking string `env:"BOOKING_COLLECTION,required" envDefault:"Booking"`
	}
}

type RabbitMqConfig struct {
	Username  Secret `env:"RABBITMQ_USERNAME,required"`
	Password  Secret `env:"RABBITMQ_PASSWORD,required"`
	Host      string `env:"RABBITMQ_HOST,required"`
	Port      int    `env:"RABBITMQ_PORT,required"`
	Producers struct {
		Cleaning RabbitMqProducerConfig `envPrefix:"CLEANING_"`
	}
	Consumers struct {
		HourChange RabbitMqConsumerConfig `envPrefix:"HOUR_CHANGE_"`
		DayChange  RabbitMqConsumerConfig `envPrefix:"DAY_CHANGE_"`
	}
}

type RedisConfig struct {
	Username Secret `env:"REDIS_USERNAME" envDefault:""`
	Password Secret `env:"REDIS_PASSWORD" envDefault:""`
	Host     string `env:"REDIS_HOST,required" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT,required" envDefault:"6379"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

type TelemetryConfig struct {
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	OTLPGrpcPort   int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT,required"`
	OTLPHealthPort int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT,required"`
	OTLPInsecure   bool   `env:"OTEL_EXPORTER_OTLP_INSECURE,required"`
}

func LoadConfigs() (Configs, error) {
	var cfg Configs
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	slog.InfoContext(context.Background(), "config loaded", "config", cfg)

	return cfg, nil
}
