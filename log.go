package main

import (
	"github.com/caarlos0/env"
	"go.uber.org/zap"
)

var logger zap.Logger
var log *zap.SugaredLogger

type logConfig struct {
	Debug bool `env:"DEBUG"`
}

func init() {
	var cfg logConfig
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Unable configure logger: \n", err)
	}
	var config zap.Config
	if cfg.Debug {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	logger, err := config.Build()
	if err != nil {
		panic("Unable to create logger")
	}
	log = logger.Sugar()
}
