package ioc

import (
	"github.com/gotomicro/ego/core/econf"
	"github.com/patrickmn/go-cache"
	"time"
)

func InitGoCache() *cache.Cache {
	type Config struct {
		DefaultExpiration time.Duration `yaml:"defaultExpiration"`
		CleanupInterval   time.Duration `yaml:"cleanupInterval"`
	}
	var cfg Config
	err := econf.UnmarshalKey("cache", &cfg)
	if err != nil {
		panic(err)
	}
	c := cache.New(cfg.DefaultExpiration, cfg.CleanupInterval)
	return c
}
