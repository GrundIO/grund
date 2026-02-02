package shared

import (
	"github.com/Saturn-Fintech/grund/internal/application/wiring"
	"github.com/Saturn-Fintech/grund/internal/config"
)

var (
	// Container is the DI container initialized by root command
	Container *wiring.Container

	// ConfigResolver is the config resolver initialized by root command
	ConfigResolver *config.ConfigResolver
)
