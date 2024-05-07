package mock

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/winkor4/taktaev_project_sp56/internal/pkg/config"
)

type mockServer struct {
	cfg *config.Config
	db  *mockDB
}

// Мок сервер эмулирует работы сервиса расчетов бонусов
// Запускается только во время тестов
func Run(cfg *config.Config) {

	dsn := "host=localhost user=postgres password=123 dbname=gophermart sslmode=disable"

	db, err := newDB(dsn)
	if err != nil {
		return
	}

	srv := mockServer{
		cfg: cfg,
		db:  db,
	}

	db.truncate()
	go calculateAccural(db)

	adr := strings.ReplaceAll(cfg.AccrualSystemAddress, "http://", "")

	err = http.ListenAndServe(adr, srvRouter(&srv))
	if err != nil {
		return
	}
}

func srvRouter(s *mockServer) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/api/orders/{number}", getOrder(s))

	return r
}

func getOrder(s *mockServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		order := chi.URLParam(r, "number")
		if order == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		accuralData, err := s.db.getAccrual(order)
		if err != nil && err != sql.ErrNoRows {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(accuralData); err != nil {
			http.Error(w, "Can't encode response", http.StatusInternalServerError)
			return
		}
	}
}

func calculateAccural(db *mockDB) {
	for {
		err := db.newOrders()
		if err != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
}
