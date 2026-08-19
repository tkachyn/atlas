package server

import (
	"context"
	"log"
)

// server owns atlas's long-running process
//
// networking will be added in milestone 2. for now, run keeps the process
// alive until its context is cancelled, which gives the future tcp server
// the same lifecycle shape
type Server struct {
	addr string
}

func New(addr string) *Server {
	return &Server{addr: addr}
}

// run starts the server and blocks until ctx is cancelled
func (s *Server) Run(ctx context.Context) error {
	log.Printf("Atlas server starting...")
	log.Printf("Listening on %s", s.addr)

	<-ctx.Done()

	log.Printf("Atlas server shutting down...")
	return nil
}
