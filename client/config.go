package client

import (
	"os"
	"time"

	"github.com/jinzhu/configor"
)

// Config holds the config.
type Config struct {
	PrivateKey    string
	MnemonicWords string
	Algod         struct {
		Endpoint  string
		APIToken  string
		UserAgent string
	}
	Account2 struct {
		PrivateKey    string
		MnemonicWords string
	}
	Indexer struct {
		Endpoint string
		APIToken string
	}
	APPID       uint64
	ReignPeriod time.Duration
}

// NewConfig returns a new configuration struct.
func NewConfig() (*Config, error) {
	var cfg Config

	err := configor.
		New(&configor.Config{
			ENVPrefix:   "KOA",
			Environment: os.Getenv("ENVIRONMENT"),
		}).
		Load(&cfg, "./configs/config.yml")

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
