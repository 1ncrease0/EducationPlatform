package server

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	notify     chan error
}

func New(address string, timeout time.Duration, idleTimeout time.Duration, handler http.Handler) *Server {
	httpServer := &http.Server{
		Addr:         address,
		Handler:      handler,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  idleTimeout,
	}

	s := &Server{httpServer: httpServer}
	return s
}

func (s *Server) Start() {
	go func() {
		s.notify <- s.httpServer.ListenAndServe()
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
