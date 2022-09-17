package service

import (
	"context"
	"io"

	"github.com/tahmooress/wallet-manager/entities"
)

type Usecases interface {
	AddBalance(voucher *entities.Voucher) error
	GetBalance(ctx context.Context, mobile string) (*entities.Account, error)
	io.Closer
}
