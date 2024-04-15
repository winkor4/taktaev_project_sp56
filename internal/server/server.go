package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/winkor4/taktaev_project_sp56/internal/pkg/config"
	"github.com/winkor4/taktaev_project_sp56/internal/storage"
)

type Config struct {
	Cfg *config.Config
	DB  *storage.DB
}

type session struct {
	user string
}
type Server struct {
	cfg     *config.Config
	db      *storage.DB
	session session
}

var jwtKey = []byte("secret_key")

type Claims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

func New(cfg Config) *Server {
	return &Server{
		cfg: cfg.Cfg,
		db:  cfg.DB,
	}
}

func (s *Server) Run() error {
	go Workers(s)
	log.Println("start server " + s.cfg.RunAddress)
	return http.ListenAndServe(s.cfg.RunAddress, SrvRouter(s))
}

func SrvRouter(s *Server) *chi.Mux {
	r := chi.NewRouter()

	r.Post("/api/user/register", checkContentTypeMiddleware(register(s), "application/json"))
	r.Post("/api/user/login", checkContentTypeMiddleware(login(s), "application/json"))
	r.Mount("/api/user", ordersRouter(s))

	return r
}

func ordersRouter(s *Server) *chi.Mux {
	r := chi.NewRouter()
	r.Use(authorizationMiddleware(s))

	r.Post("/orders", checkContentTypeMiddleware(uploadOrder(s), "text/plain"))
	r.Get("/orders", getOrders(s))
	r.Get("/balance", getBalance(s))

	return r
}

func checkContentTypeMiddleware(h http.HandlerFunc, exContentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/x-gzip") {
			r.Header.Set("Content-Type", exContentType)
			h(w, r)
			return
		}

		if !strings.Contains(contentType, exContentType) {
			http.Error(w, "unexpected Content-Type", http.StatusBadRequest)
			return
		}
		h(w, r)
	}
}

func authorizationMiddleware(s *Server) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			c, err := r.Cookie("token")
			if err != nil {
				if err == http.ErrNoCookie {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "can't read cookie", http.StatusBadRequest)
				return
			}

			tokenStr := c.Value
			claims := new(Claims)

			tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				return jwtKey, nil
			})

			if err != nil {
				if err == jwt.ErrSignatureInvalid {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "can't parse cookie", http.StatusBadRequest)
				return
			}
			if !tkn.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !s.db.Authorized(claims.Login) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			s.session.user = claims.Login

			h.ServeHTTP(w, r)
		})
	}
}

func Workers(s *Server) {
	go refreshOrders(s)
}

func refreshOrders(s *Server) {
	if s.cfg.AccuralSystemAddress == "" {
		return
	}
	for {
		orders, err := s.db.OrdersToRefresh()
		if err != nil {
			break
		}
		if len(orders) > 0 {
			err := getOrdersAccrual(s, orders)
			if err != nil {
				break
			}
		}
		time.Sleep(time.Second * 2)
	}
}
