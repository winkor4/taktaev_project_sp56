package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/winkor4/taktaev_project_sp56/internal/model"
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

type ctxKey string

const keyUser ctxKey = "user"

// Осознанно оставил ключ в коде
// Куда его лучше прятать, нужна подсказка
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

//  1. Текст ТЗ: "клиент может поддерживать HTTP-запросы/ответы со сжатием данных"
//     Если это требуется, доделаю. Только вопрос какие типы сжатия нужно поддерживать?
//  2. Не подключал логгер, если требуется сделаю. Какую информацию принято логировать?
func (s *Server) Run() error {
	Workers(s)
	return http.ListenAndServe(s.cfg.RunAddress, SrvRouter(s))
}

func SrvRouter(s *Server) *chi.Mux {
	r := chi.NewRouter()
	r.Use(gzipMiddleware)

	r.Post("/api/user/register", checkContentTypeMiddleware(register(s), "application/json"))
	r.Post("/api/user/login", checkContentTypeMiddleware(login(s), "application/json"))
	r.Mount("/api/user", ordersRouter(s))

	return r
}

func ordersRouter(s *Server) *chi.Mux {
	r := chi.NewRouter()
	r.Use(authorizationMiddleware())

	r.Post("/orders", checkContentTypeMiddleware(uploadOrder(s), "text/plain"))
	r.Get("/orders", getOrders(s))
	r.Get("/balance", getBalance(s))
	r.Post("/balance/withdraw", checkContentTypeMiddleware(withdrawBonuses(s), "application/json"))
	r.Get("/withdrawals", getWithdrawals(s))

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

func authorizationMiddleware() func(h http.Handler) http.Handler {
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

			ctx := context.WithValue(r.Context(), keyUser, claims.Login)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Workers(s *Server) {
	go refreshOrders(s)
}

type chAccrualSystem struct {
	chOrders  chan string
	chAccrual chan model.AccrualSchema
	closed    bool
	err       error
}

func (ch *chAccrualSystem) close(err error) {
	ch.closed = true
	ch.err = err
}

func refreshOrders(s *Server) {
	log.Println("start refreshOrders with dsn: " + s.cfg.AccrualSystemAddress)
	var ch chAccrualSystem
	ch.chOrders = make(chan string)
	ch.chAccrual = make(chan model.AccrualSchema)
	ch.closed = false

	defer close(ch.chOrders)
	defer close(ch.chAccrual)

	for i := 0; i < 10; i++ {
		go getOrdersFromAccrualSystem(s, &ch)
	}
	go updateOrders(s, &ch)

	for {

		if ch.closed {
			log.Println("stop refreshOrders: " + ch.err.Error())
			break
		}

		orders, err := s.db.OrdersToRefresh(context.Background())
		if err != nil {
			log.Println("stop refreshOrders: " + err.Error())
			ch.close(err)
			break
		}

		for _, order := range orders {
			ch.chOrders <- order
		}
		time.Sleep(time.Second * 2)
	}
}

func getOrdersFromAccrualSystem(s *Server, ch *chAccrualSystem) {
	basePath := s.cfg.AccrualSystemAddress + "/api/orders/"
	client := http.Client{}

	for order := range ch.chOrders {

		request, err := http.NewRequest(http.MethodGet, basePath+order, nil)
		if err != nil {
			ch.close(err)
			return
		}
		r, err := client.Do(request)
		if err != nil {
			ch.close(err)
			return
		}
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			ch.close(err)
			return
		}
		err = r.Body.Close()
		if err != nil {
			ch.close(err)
			return
		}
		if r.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Second * 2)
			continue
		}
		if r.StatusCode != http.StatusOK {
			time.Sleep(time.Second * 2)
			continue
		}
		var accrualData model.AccrualSchema
		err = json.Unmarshal(rBody, &accrualData)
		if err != nil {
			ch.close(err)
			return
		}
		if ch.closed {
			break
		}
		ch.chAccrual <- accrualData
	}
}

func updateOrders(s *Server, ch *chAccrualSystem) {
	ctx := context.Background()

	for data := range ch.chAccrual {
		accrualList := make([]model.AccrualSchema, 0)
		accrualList = append(accrualList, data)

		err := s.db.UpdateOrders(ctx, accrualList)
		if err != nil {
			ch.close(err)
			return
		}
	}

}
