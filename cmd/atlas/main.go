package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/styro/atlas/internal/config"
	"github.com/styro/atlas/internal/server"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg.Addr)
	if err := srv.Run(ctx); err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
