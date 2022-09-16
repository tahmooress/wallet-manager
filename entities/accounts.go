package entities

import "time"

type Account struct {
	User      string
	Balance   int64
	CreatedAt time.Time
}

type Voucher struct {
	User     string `json:"user"`
	Campaign string `json:"campaign"`
	Value    int64  `json:"value"`
}
