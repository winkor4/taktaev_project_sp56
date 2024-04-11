package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/winkor4/taktaev_project_sp56/internal/pkg/config"
	"github.com/winkor4/taktaev_project_sp56/internal/storage"
)

type Config struct {
	Cfg *config.Config
	DB  *storage.DB
}
type Server struct {
	cfg *config.Config
	db  *storage.DB
}

func New(cfg Config) *Server {
	return &Server{
		cfg: cfg.Cfg,
		db:  cfg.DB,
	}
}

func (s *Server) Run() error {
	return http.ListenAndServe(s.cfg.RunAddress, SrvRouter(s))
}

func SrvRouter(s *Server) *chi.Mux {
	r := chi.NewRouter()

	return r
}
