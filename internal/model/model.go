package model

import "time"

type RegisterSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type OrderSchema struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadedAt string `json:"uploaded_at"`
	Date       time.Time
}

type AccrualSchema struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

type BalaneSchema struct {
	Current   int `json:"current"`
	WithDrawn int `json:"withdrawn"`
}
