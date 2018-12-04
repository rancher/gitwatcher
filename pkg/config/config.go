package config

import (
	"context"
	"github.com/rancher/types/config"
	"sync"
)

type Config struct {
	sync.Mutex
	Management *config.ScaledContext
	ctx        context.Context
	cancel     context.CancelFunc
}
