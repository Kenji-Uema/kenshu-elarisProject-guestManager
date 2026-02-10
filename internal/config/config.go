package config

import (
	"errors"

	"github.com/caarlos0/env/v11"
)

type Configs struct {
	AppConfig
	MongoConfig
	RabbitMqConfig
	ClockEmuConfig
	GuestCollectionConfig
	CottageCollectionConfig
	BookingCollectionConfig
	CleaningExchangeConfig
	TelemetryConfig
}

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME"`
	Version     string `env:"VERSION"`
	Host        string `env:"SERVICE_HOST,required"`
	Port        int    `env:"SERVICE_PORT,required"`
}

type MongoConfig struct {
	Username string `env:"MONGO_INITDB_ROOT_USERNAME,required"`
	Password string `env:"MONGO_INITDB_ROOT_PASSWORD,required"`
	Host     string `env:"MONGO_HOST,required"`
	Database string `env:"MONGO_DATABASE,required"`
}

type RabbitMqConfig struct {
	Username string `env:"RABBITMQ_USERNAME,required"`
	Password string `env:"RABBITMQ_PASSWORD,required"`
	Host     string `env:"RABBITMQ_HOST,required"`
	Port     int    `env:"RABBITMQ_PORT,required"`
}

type ClockEmuConfig struct {
	GrpcHost string `env:"CLOCK_EMU_GRPC_HOST,required"`
	GrpcPort int    `env:"CLOCK_EMU_GRPC_PORT,required"`
}

type GuestCollectionConfig struct {
	Name string `env:"GUEST_COLLECTION,required" envDefault:"Guest"`
}

type CottageCollectionConfig struct {
	Name string `env:"COTTAGE_COLLECTION,required" envDefault:"Cottage"`
}

type BookingCollectionConfig struct {
	Name string `env:"BOOKING_COLLECTION,required" envDefault:"Booking"`
}

type CleaningExchangeConfig struct {
	Name           string `env:"CLEANING_EXCHANGE,required"`
	Kind           string `env:"CLEANING_EXCHANGE_KIND,required"`
	Durable        bool   `env:"CLEANING_EXCHANGE_DURABLE,required"`
	AutoDelete     bool   `env:"CLEANING_EXCHANGE_AUTO_DELETE,required"`
	Internal       bool   `env:"CLEANING_EXCHANGE_INTERNAL,required"`
	NoWait         bool   `env:"CLEANING_EXCHANGE_NO_WAIT,required"`
	ConfirmNotWait bool   `env:"RABBITMQ_CONFIRM_NOT_WAIT,required"`
}

type TelemetryConfig struct {
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	OTLPGrpcPort   int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT,required"`
	OTLPHealthPort int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT,required"`
	OTLPInsecure   bool   `env:"OTEL_EXPORTER_OTLP_INSECURE,required"`
}

func LoadConfigs() (Configs, error) {
	var err error

	appConfig, loadErr := loadConfig[AppConfig]()
	err = errors.Join(err, loadErr)
	mongoConfig, loadErr := loadConfig[MongoConfig]()
	err = errors.Join(err, loadErr)
	rabbitmqConfig, loadErr := loadConfig[RabbitMqConfig]()
	err = errors.Join(err, loadErr)
	clockEmuConfig, loadErr := loadConfig[ClockEmuConfig]()
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
		clockEmuConfig,
		guestCollectionConfig,
		cottageCollectionConfig,
		bookingCollectionConfig,
		cleaningExchangeConfig,
		telemetryConfig,
	}, err
}

func loadConfig[C AppConfig | MongoConfig | GuestCollectionConfig | CottageCollectionConfig | ClockEmuConfig |
	BookingCollectionConfig | RabbitMqConfig | CleaningExchangeConfig | TelemetryConfig]() (C, error) {
	var c C
	if err := env.Parse(&c); err != nil {
		return c, err
	}

	return c, nil
}
