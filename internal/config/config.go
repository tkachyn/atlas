package config

import (
	"flag"
	"fmt"
)

// config contains the settings that atlas needs to start
type Config struct {
	Addr        string
	DataFile    string
	MaxLogBytes int64
}

// parse reads command line flags, then returns server config
func Parse(args []string) (Config, error) {
	flags := flag.NewFlagSet("atlas", flag.ContinueOnError)
	addr := flags.String("addr", ":6379", "TCP address Atlas will listen on")
	dataFile := flags.String("data-file", "atlas.aof", "append-only persistence file")
	maxLogBytes := flags.Int64("max-log-bytes", 64*1024*1024, "compact persistence log after this size")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	return Config{Addr: *addr, DataFile: *dataFile, MaxLogBytes: *maxLogBytes}, nil
}
