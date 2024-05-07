package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

var (
	flagRunAddress           string
	flagDatabaseURI          string
	flagAccuralSystemAddress string
)

func Parse() (*Config, error) {

	cfg := new(Config)

	flag.StringVar(&flagRunAddress, "a", "", "srv run addres and port")
	flag.StringVar(&flagDatabaseURI, "d", "", "PostgresSQL server")
	flag.StringVar(&flagAccuralSystemAddress, "r", "", "accural system address and port")
	flag.Parse()

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.RunAddress == "" {
		cfg.RunAddress = flagRunAddress
	}
	if cfg.DatabaseURI == "" {
		cfg.DatabaseURI = flagDatabaseURI
	}
	if cfg.AccrualSystemAddress == "" {
		cfg.AccrualSystemAddress = flagAccuralSystemAddress
	}

	return cfg, nil

}
