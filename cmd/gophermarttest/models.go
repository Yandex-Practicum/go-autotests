package main

import (
	"time"
)

type userBalance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float32   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type accrualOrder struct {
	Order string             `json:"order"`
	Goods []accrualOrderGood `json:"goods"`
}

type accrualOrderGood struct {
	Description string  `json:"description"`
	Price       float32 `json:"price"`
}

type orderAccrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}

type userWithdrawal struct {
	Order       string    `json:"order"`
	Sum         float32   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
