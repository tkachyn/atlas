package config

import (
	"flag"
	"fmt"
)

// config contains the settings that atlas needs to start
type Config struct {
	Addr string
}

// parse reads command line flags, then returns server config
func Parse(args []string) (Config, error) {
	flags := flag.NewFlagSet("atlas", flag.ContinueOnError)
	addr := flags.String("addr", ":6379", "TCP address Atlas will listen on")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	return Config{Addr: *addr}, nil
}
