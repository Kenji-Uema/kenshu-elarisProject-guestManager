package config

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type MongoDbConfig struct {
	Url      string `env:"MONGO_URL,required"`
	Port     string `env:"MONGO_PORT,required"`
	Db       string `env:"MONGO_DB,required"`
	User     string `env:"MONGO_USER,required"`
	Password string `env:"MONGO_PASSWORD,required"`
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

func LoadConfig[C MongoDbConfig | GuestCollectionConfig | CottageCollectionConfig |
	BookingCollectionConfig | RabbitMqConfig | CleaningExchangeConfig]() *C {
	var c C
	if err := env.Parse(&c); err != nil {
		log.Fatal(err)
	}

	return &c
}
