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

type GuestCollectionConfig struct {
	Name string `env:"GUEST_COLLECTION" envDefault:"Guest"`
}

type CottageCollectionConfig struct {
	Name string `env:"COTTAGE_COLLECTION" envDefault:"Cottage"`
}

type BookingCollectionConfig struct {
	Name string `env:"BOOKING_COLLECTION" envDefault:"Booking"`
}

func LoadConfig[C MongoDbConfig | GuestCollectionConfig | CottageCollectionConfig |
	BookingCollectionConfig]() *C {
	var c C
	if err := env.Parse(&c); err != nil {
		log.Fatal(err)
	}

	return &c
}
