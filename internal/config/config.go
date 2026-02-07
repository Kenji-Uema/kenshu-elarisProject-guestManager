package config

import (
	"errors"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Configs struct {
	AppConfig
	MongoConfig
	RabbitMqConfig
	GuestCollectionConfig
	CottageCollectionConfig
	BookingCollectionConfig
	CleaningExchangeConfig
	TelemetryConfig
}

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME" required:"true"`
	Version     string `env:"VERSION" required:"true"`
}

type MongoConfig struct {
	Username string `env:"MONGO_INITDB_ROOT_USERNAME" required:"true"`
	Password string `env:"MONGO_INITDB_ROOT_PASSWORD" required:"true"`
	Host     string `env:"MONGO_HOST" required:"true"`
	Database string `env:"MONGO_DATABASE" required:"true"`
}

type RabbitMqConfig struct {
	Url string `env:"RABBITMQ_URL,required"`
}

type GuestCollectionConfig struct {
	Name string `env:"GUEST_COLLECTION" envDefault:"Guest"`
}

type CottageCollectionConfig struct {
	Name string `env:"COTTAGE_COLLECTION" envDefault:"Cottage"`
}

type BookingCollectionConfig struct {
	Name string `env:"BOOKING_COLLECTION" envDefault:"Booking"`
}

type CleaningExchangeConfig struct {
	Name           string `env:"CLEANING_EXCHANGE" envDefault:"ex.cleaning"`
	Kind           string `env:"CLEANING_EXCHANGE_KIND" envDefault:"direct"`
	Durable        bool   `env:"CLEANING_EXCHANGE_DURABLE" envDefault:"true"`
	AutoDelete     bool   `env:"CLEANING_EXCHANGE_AUTO_DELETE" envDefault:"false"`
	Internal       bool   `env:"CLEANING_EXCHANGE_INTERNAL" envDefault:"false"`
	NoWait         bool   `env:"CLEANING_EXCHANGE_NO_WAIT" envDefault:"false"`
	ConfirmNotWait bool   `env:"RABBITMQ_CONFIRM_NOT_WAIT" envDefault:"false"`
}

type TelemetryConfig struct {
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" required:"true"`
	OTLPGrpcPort   int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT" required:"true"`
	OTLPHealthPort int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT" required:"true"`
	OTLPInsecure   bool   `env:"OTEL_EXPORTER_OTLP_INSECURE" required:"true"`
}

func LoadConfigs() (Configs, error) {
	var err error

	appConfig, loadErr := loadConfig[AppConfig]()
	err = errors.Join(err, loadErr)
	mongoConfig, loadErr := loadConfig[MongoConfig]()
	err = errors.Join(err, loadErr)
	rabbitmqConfig, loadErr := loadConfig[RabbitMqConfig]()
	err = errors.Join(err, loadErr)
	guestCollectionConfig, loadErr := loadConfig[GuestCollectionConfig]()
	err = errors.Join(err, loadErr)
	cottageCollectionConfig, loadErr := loadConfig[CottageCollectionConfig]()
	err = errors.Join(err, loadErr)
	bookingCollectionConfig, loadErr := loadConfig[BookingCollectionConfig]()
	err = errors.Join(err, loadErr)
	cleaningExchangeConfig, loadErr := loadConfig[CleaningExchangeConfig]()
	err = errors.Join(err, loadErr)
	telemetryConfig, loadErr := loadConfig[TelemetryConfig]()
	err = errors.Join(err, loadErr)

	return Configs{
		appConfig,
		mongoConfig,
		rabbitmqConfig,
		guestCollectionConfig,
		cottageCollectionConfig,
		bookingCollectionConfig,
		cleaningExchangeConfig,
		telemetryConfig,
	}, err
}

func loadConfig[C AppConfig | MongoConfig | GuestCollectionConfig | CottageCollectionConfig |
	BookingCollectionConfig | RabbitMqConfig | CleaningExchangeConfig | TelemetryConfig]() (C, error) {
	var c C
	if err := env.Parse(&c); err != nil {
		slog.Error("parse env config", "error", err)
		return c, err
	}

	return c, nil
}
