package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/winkor4/taktaev_project_sp56/internal/model"
	"github.com/winkor4/taktaev_project_sp56/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

func register(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var schema model.RegisterSchema
		err := json.NewDecoder(r.Body).Decode(&schema)
		if err != nil {
			http.Error(w, "Can't read body", http.StatusBadRequest)
			return
		}

		if schema.Login == "" || schema.Password == "" {
			http.Error(w, "empty login/password", http.StatusBadRequest)
			return
		}

		hash, err := hashPassword(schema.Password)
		if err != nil {
			http.Error(w, "can't generate hash from password", http.StatusInternalServerError)
			return
		}

		conflict, err := s.db.Register(r.Context(), schema.Login, hash)
		if err != nil {
			http.Error(w, "can't register", http.StatusInternalServerError)
			return
		}
		if conflict {
			http.Error(w, "login not unique", http.StatusConflict)
			return
		}

		s.db.Authorisation(schema.Login)

		token, err := authToken(schema.Login)
		if err != nil {
			http.Error(w, "can't auth", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, token)

		w.WriteHeader(http.StatusOK)
	}
}

func hashPassword(pass string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pass), 8)
	return string(bytes), err
}

func checkPasswordHash(pass, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
	return err == nil
}

func login(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var schema model.RegisterSchema
		err := json.NewDecoder(r.Body).Decode(&schema)
		if err != nil {
			http.Error(w, "Can't read body", http.StatusBadRequest)
			return
		}
		if schema.Login == "" {
			http.Error(w, "empty login", http.StatusBadRequest)
			return
		}

		hash, err := s.db.GetPass(r.Context(), schema.Login)
		if err != nil {
			http.Error(w, "can't auth", http.StatusInternalServerError)
			return
		}

		if !checkPasswordHash(schema.Password, hash) {
			http.Error(w, "can't auth", http.StatusUnauthorized)
			return
		}

		s.db.Authorisation(schema.Login)

		token, err := authToken(schema.Login)
		if err != nil {
			http.Error(w, "can't auth", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, token)

		w.WriteHeader(http.StatusOK)

	}
}

func authToken(login string) (*http.Cookie, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Login: login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtKey)
	if err != nil {
		return nil, err
	}

	return &http.Cookie{
		Name:    "token",
		Value:   tokenStr,
		Expires: expirationTime,
	}, nil
}

func uploadOrder(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Can't read body", http.StatusBadRequest)
			return
		}

		orderNumber := string(data)
		if orderNumber == "" {
			http.Error(w, "empty order number", http.StatusBadRequest)
			return
		}

		if badNumberFormat(orderNumber) {
			http.Error(w, "bad format order number", http.StatusUnprocessableEntity)
			return
		}

		err = s.db.CheckOrder(r.Context(), s.session.user, orderNumber)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			return
		} else if err != sql.ErrNoRows {
			if err == storage.ErrConflict {
				w.WriteHeader(http.StatusConflict)
				return
			}
			http.Error(w, "can't check order number", http.StatusInternalServerError)
			return
		}

		err = s.db.UploadOrder(r.Context(), s.session.user, orderNumber)
		if err != nil {
			http.Error(w, "can't write order number", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)

	}
}

func badNumberFormat(str string) bool {
	sum := 0
	nDigits := len(str)
	parity := nDigits % 2

	for i := 0; i < nDigits; i++ {
		digit, err := strconv.Atoi(string(str[i]))
		if err != nil {
			return true
		}
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 != 0
}

func getOrders(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
		defer cancel()
		r = r.WithContext(ctx)

		orders, err := s.db.GetOrders(r.Context(), s.session.user)
		if err != nil {
			http.Error(w, "can't get user's orders", http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			http.Error(w, "no content", http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			http.Error(w, "Can't encode response", http.StatusInternalServerError)
			return
		}
	}
}

func getOrdersAccrual(s *Server, orders []string) error {

	basePath := s.cfg.AccuralSystemAddress + "/api/orders/"

	client := http.Client{}

	accrualList := make([]model.AccrualSchema, 0)

	for _, order := range orders {
		request, err := http.NewRequest(http.MethodGet, basePath+order, nil)
		if err != nil {
			return err
		}

		r, err := client.Do(request)
		if err != nil {
			return err
		}
		rBody, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		err = r.Body.Close()
		if err != nil {
			return err
		}

		if r.StatusCode == http.StatusTooManyRequests {
			return nil
		}

		if r.StatusCode != http.StatusOK {
			continue
		}

		var accrualData model.AccrualSchema
		err = json.Unmarshal(rBody, &accrualData)
		if err != nil {
			return err
		}
		accrualList = append(accrualList, accrualData)
	}

	if len(accrualList) == 0 {
		return nil
	}

	ctx := context.Background()
	err := s.db.UpdateOrders(ctx, accrualList)
	if err != nil {
		return err
	}
	err = s.db.SetBonuses(ctx, accrualList)
	if err != nil {
		return err
	}

	return nil
}

func getBalance(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		balance, err := s.db.GetBalance(r.Context(), s.session.user)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "can't get balance", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(balance); err != nil {
			http.Error(w, "Can't encode response", http.StatusInternalServerError)
			return
		}

	}
}

func withdrawBonuses(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var schema model.WithdrawSchema
		err := json.NewDecoder(r.Body).Decode(&schema)
		if err != nil {
			http.Error(w, "Can't read body", http.StatusBadRequest)
			return
		}

		orderNumber := string(schema.Order)
		if orderNumber == "" {
			http.Error(w, "empty order number", http.StatusBadRequest)
			return
		}

		if badNumberFormat(orderNumber) {
			http.Error(w, "bad format order number", http.StatusUnprocessableEntity)
			return
		}

		err = s.db.WithdrawBonuses(r.Context(), s.session.user, schema)
		if err != nil {
			if err == storage.ErrPaymentRequired {
				http.Error(w, "not enough bonuses", http.StatusPaymentRequired)
				return
			}
			http.Error(w, "can't withdraw bonuses", http.StatusInternalServerError)
			return
		}
	}
}

func getWithdrawals(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		orders, err := s.db.Getwithdrawels(r.Context(), s.session.user)
		if err != nil {
			http.Error(w, "can't get user's orders", http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			http.Error(w, "no content", http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			http.Error(w, "Can't encode response", http.StatusInternalServerError)
			return
		}
	}
}
