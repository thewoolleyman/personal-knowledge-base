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

// TokenChecker reports the health of an authentication token.
type TokenChecker interface {
	TokenHealth() TokenStatus
}

// TokenStatus describes whether a token is valid.
type TokenStatus struct {
	Valid     bool          `json:"valid"`
	ExpiresIn time.Duration `json:"-"`
	Error     string        `json:"error,omitempty"`
}

type healthResponse struct {
	Status  string       `json:"status"`
	Version string       `json:"version"`
	Uptime  string       `json:"uptime"`
	Token   *tokenHealth `json:"token,omitempty"`
}

type tokenHealth struct {
	Valid     bool   `json:"valid"`
	ExpiresIn string `json:"expires_in,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Server struct {
	httpServer   *http.Server
	listener     net.Listener
	mux          *http.ServeMux
	startTime    time.Time
	tokenChecker TokenChecker
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

// SetTokenChecker sets an optional TokenChecker whose status is included in
// the /health response. Pass nil to clear.
func (s *Server) SetTokenChecker(tc TokenChecker) {
	s.tokenChecker = tc
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime).Truncate(time.Second).String()
	resp := healthResponse{
		Status:  "ok",
		Version: Version,
		Uptime:  uptime,
	}

	if s.tokenChecker != nil {
		ts := s.tokenChecker.TokenHealth()
		th := &tokenHealth{
			Valid: ts.Valid,
		}
		if ts.Valid {
			th.ExpiresIn = ts.ExpiresIn.String()
		} else {
			resp.Status = "degraded"
			th.Error = ts.Error
		}
		resp.Token = th
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
