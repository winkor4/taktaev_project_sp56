package app

import (
	"github.com/winkor4/taktaev_project_sp56/internal/pkg/config"
	"github.com/winkor4/taktaev_project_sp56/internal/server"
	"github.com/winkor4/taktaev_project_sp56/internal/storage"
)

func Run() error {

	cfg, err := config.Parse()
	if err != nil {
		return err
	}

	db, err := storage.New(cfg.DatabaseURI)
	if err != nil {
		return err
	}

	srv := server.New(server.Config{
		Cfg: cfg,
		DB:  db,
	})

	return srv.Run()

}
