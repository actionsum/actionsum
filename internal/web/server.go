package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/actionsum/actionsum/internal/config"
	"github.com/actionsum/actionsum/internal/database"
)

type Server struct {
	config  *config.Config
	handler *Handler
	server  *http.Server
}

func NewServer(cfg *config.Config, repo *database.Repository, customPort int) *Server {
	handler := NewHandler(cfg, repo)
	mux := http.NewServeMux()
	handler.SetupRoutes(mux)

	port := cfg.Web.Port
	if customPort > 0 {
		port = customPort
	}

	addr := fmt.Sprintf("%s:%d", cfg.Web.Host, port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		config:  cfg,
		handler: handler,
		server:  httpServer,
	}
}

func (s *Server) Start() error {
	log.Printf("Starting web server on http://%s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down web server...")
	return s.server.Shutdown(ctx)
}

func (s *Server) GetAddress() string {
	return s.server.Addr
}
