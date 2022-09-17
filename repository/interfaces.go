package repository

import (
	"context"
	"io"

	"github.com/tahmooress/wallet-manager/entities"
)

type Tx interface {
	AddBalance(voucher *entities.Voucher) error
}

type Reader interface {
	GetBalance(ctx context.Context, moblie string) (*entities.Account, error)
}

type Writer interface {
	ExecWrite(func(tx Tx) error) error
}

type DB interface {
	Writer
	Reader
	io.Closer
}
