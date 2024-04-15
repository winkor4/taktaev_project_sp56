package model

import "time"

type RegisterSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type OrderSchema struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual"`
	UploadedAt string    `json:"uploaded_at"`
	Date       time.Time `json:"-"`
}

type AccrualSchema struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

type BalaneSchema struct {
	Current   float32 `json:"current"`
	WithDrawn float32 `json:"withdrawn"`
}
