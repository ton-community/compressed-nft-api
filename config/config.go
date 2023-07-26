package config

import (
	"github.com/caarlos0/env/v9"
	"github.com/joho/godotenv"
)

var Config = struct {
	Database      string `env:"POSTGRES_URI"`
	Port          int    `env:"PORT"`
	AdminUsername string `env:"ADMIN_USERNAME"`
	AdminPassword string `env:"ADMIN_PASSWORD"`
	Depth         int    `env:"DEPTH"`
	DataDir       string `env:"DATA_DIR"`
	Toncenter     string `env:"TONCENTER_URI"`
}{}

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	if err := env.Parse(&Config); err != nil {
		panic(err)
	}
}
