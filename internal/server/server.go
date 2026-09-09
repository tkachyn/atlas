package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/tkachyn/atlas/internal/command"
	"github.com/tkachyn/atlas/internal/persistence"
	"github.com/tkachyn/atlas/internal/protocol"
	"github.com/tkachyn/atlas/internal/store"
)

// server owns atlas's tcp listener and active client connections
type Server struct {
	addr        string
	mu          sync.Mutex
	listener    net.Listener
	clients     map[net.Conn]struct{}
	clientWg    sync.WaitGroup
	data        *store.Store
	logFile     *persistence.Log
	maxLogBytes int64
}

func New(addr string) *Server {
	return NewWithPersistence(addr, store.New(), nil, 0)
}

// create a server that records successful commands
func NewWithPersistence(addr string, data *store.Store, logFile *persistence.Log, maxLogBytes int64) *Server {
	return &Server{
		addr:        addr,
		clients:     make(map[net.Conn]struct{}),
		data:        data,
		logFile:     logFile,
		maxLogBytes: maxLogBytes,
	}
}

// address returns the address currently used by the listener
func (s *Server) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}

	return s.addr
}

// run starts the tcp server and blocks until ctx is cancelled
func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	log.Printf("Atlas server starting...")
	log.Printf("Listening on %s", listener.Addr())

	// closing the listener unblocks accept when shutdown is requested
	stopListener := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.closeListener()
		case <-stopListener:
		}
	}()

	// handle clients independently so a slow connection cannot block new clients
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}

			close(stopListener)
			s.closeConnections()
			s.clientWg.Wait()
			return fmt.Errorf("accept connection: %w", err)
		}

		s.addConnection(conn)
		s.clientWg.Add(1)
		go s.handleConnection(conn)
	}

	close(stopListener)
	// close clients before waiting so blocked read loops can finish
	s.closeConnections()
	s.clientWg.Wait()

	s.mu.Lock()
	s.listener = nil
	s.mu.Unlock()

	log.Printf("Atlas server shutting down...")
	return nil
}

func (s *Server) addConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[conn] = struct{}{}
}

func (s *Server) removeConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, conn)
}

func (s *Server) closeListener() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("close listener: %v", err)
		}
	}
}

func (s *Server) closeConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for conn := range s.clients {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("close client connection: %v", err)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.clientWg.Done()
	defer s.removeConnection(conn)
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("close client connection: %v", err)
		}
	}()

	log.Printf("client connected: %s", conn.RemoteAddr())

	writer := bufio.NewWriter(conn)
	if _, err := writer.WriteString("Atlas ready\n"); err != nil {
		log.Printf("write greeting to %s: %v", conn.RemoteAddr(), err)
		return
	}
	if err := writer.Flush(); err != nil {
		log.Printf("flush greeting to %s: %v", conn.RemoteAddr(), err)
		return
	}

	reader := bufio.NewReader(conn)
	// newline terminates requests in atlas's text protocol
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			response := s.processRequest(line)
			if _, writeErr := writer.WriteString(response); writeErr != nil {
				log.Printf("write response to %s: %v", conn.RemoteAddr(), writeErr)
				return
			}
			if flushErr := writer.Flush(); flushErr != nil {
				log.Printf("flush response to %s: %v", conn.RemoteAddr(), flushErr)
				return
			}
		}

		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Printf("read from %s: %v", conn.RemoteAddr(), err)
			}
			break
		}
	}

	log.Printf("client disconnected: %s", conn.RemoteAddr())
}

func (s *Server) processRequest(line string) string {
	cmd, err := protocol.Parse(line)
	if err != nil {
		return protocol.Error(err.Error())
	}
	if cmd.Name == "EXPIREAT" {
		return protocol.Error(`unknown command "EXPIREAT"`)
	}

	if s.logFile != nil {
		// persist before responding so successful commands have a durable record
		if err := s.logFile.Append(cmd); err != nil {
			return protocol.Error("persistence failure")
		}
	}

	response := command.Execute(cmd, s.data)
	if s.logFile != nil {
		if err := s.logFile.MaybeCompact(s.maxLogBytes, s.data); err != nil {
			log.Printf("persistence compaction error: %v", err)
		}
	}

	return response
}
