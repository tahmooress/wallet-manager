package service

import (
	"context"

	"github.com/tahmooress/wallet-manager/entities"
	"github.com/tahmooress/wallet-manager/repository"
)

func (s *service) AddBalance(voucher *entities.Voucher) error {
	err := s.db.ExecWrite(func(tx repository.Tx) error {
		return tx.AddBalance(voucher)
	})
	if err != nil {
		s.logger.Errorf("service: AddBalance() >> %w", err)

		return err
	}

	return nil
}

func (s *service) GetBalance(ctx context.Context, user string) (*entities.Account, error) {
	return s.db.GetBalance(ctx, user)
}
