package server

import (
	"context"
	"errors"
	"net"
	"net/http"
)

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	mux        *http.ServeMux
}

func New(addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		mux: mux,
	}
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
