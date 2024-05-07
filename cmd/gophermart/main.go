package main

import (
	"log"

	"github.com/winkor4/taktaev_project_sp56/internal/pkg/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
