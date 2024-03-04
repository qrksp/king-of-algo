package integration

import (
	"os"

	"github.com/jinzhu/configor"
)

type config struct {
	SandboxRepoPath string
	AlgodEndpoint   string
	AlgodToken      string
}

// newConfig returns a new configuration struct.
func newConfig() (*config, error) {
	var cfg config

	err := configor.
		New(&configor.Config{
			ENVPrefix:   "INTEGRATION",
			Environment: os.Getenv("ENVIRONMENT"),
		}).
		Load(&cfg, "../configs/integration.yml")

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
