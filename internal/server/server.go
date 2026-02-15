package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"
)

// Version is set by the caller before creating a server. Defaults to "dev".
var Version = "dev"

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	mux        *http.ServeMux
	startTime  time.Time
}

func New(addr string) *Server {
	s := &Server{
		startTime: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.healthHandler)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	s.mux = mux
	return s
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime).Truncate(time.Second).String()
	resp := map[string]string{
		"status":  "ok",
		"version": Version,
		"uptime":  uptime,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Handle registers an additional HTTP handler on the server's mux.
// Must be called before Serve.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// Listen binds the socket. Must be called before Serve.
func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}
	s.listener = ln
	return nil
}

// Serve starts accepting connections. Blocks until shutdown.
// Caller must call Listen first.
func (s *Server) Serve() error {
	if s.listener == nil {
		return errors.New("must call Listen before Serve")
	}
	return s.httpServer.Serve(s.listener)
}

func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
