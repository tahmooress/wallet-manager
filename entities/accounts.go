package entities

import "time"

type Account struct {
	Mobile    string
	Balance   int64
	CreatedAt time.Time
}

type Voucher struct {
	Mobile   string `json:"mobile"`
	Campaign string `json:"campaign"`
	Value    int64  `json:"value"`
}
