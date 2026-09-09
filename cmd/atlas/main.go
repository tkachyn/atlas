package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tkachyn/atlas/internal/config"
	"github.com/tkachyn/atlas/internal/persistence"
	"github.com/tkachyn/atlas/internal/server"
	"github.com/tkachyn/atlas/internal/store"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(2)
	}

	data := store.New()
	logFile, err := persistence.Open(cfg.DataFile)
	if err != nil {
		log.Printf("persistence error: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("persistence close error: %v", err)
		}
	}()

	if err := logFile.Replay(data); err != nil {
		log.Printf("recovery error: %v", err)
		if closeErr := logFile.Close(); closeErr != nil {
			log.Printf("persistence close error: %v", closeErr)
		}
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.NewWithPersistence(cfg.Addr, data, logFile, cfg.MaxLogBytes)
	if err := srv.Run(ctx); err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
