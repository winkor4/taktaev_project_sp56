package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("secret_key")

type Claims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

func register(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var schema registerSchema
		err := json.NewDecoder(r.Body).Decode(&schema)
		if err != nil {
			http.Error(w, "Can't read body", http.StatusBadRequest)
			return
		}

		if schema.Login == "" || schema.Password == "" {
			http.Error(w, "empty login/password", http.StatusBadRequest)
			return
		}

		conflict, err := s.db.Register(schema.Login, schema.Password)
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

func login(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var schema registerSchema
		err := json.NewDecoder(r.Body).Decode(&schema)
		if err != nil {
			http.Error(w, "Can't read body", http.StatusBadRequest)
			return
		}
		if schema.Login == "" {
			http.Error(w, "empty login", http.StatusBadRequest)
			return
		}

		expPass, err := s.db.GetPass(schema.Login)
		if err != nil {
			http.Error(w, "can't auth", http.StatusInternalServerError)
			return
		}

		if schema.Password != expPass {
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
