package repository

import (
	"context"
	"fmt"

	"github.com/tahmooress/wallet-manager/entities"
)

func (r *repo) GetBalance(ctx context.Context, mobile string) (*entities.Account, error) {
	query := `SELECT * FROM accounts WHERE mobile = $1`

	var account entities.Account

	err := r.db.QueryRowContext(ctx, query, mobile).Scan(
		&account.Mobile, &account.Balance, &account.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository >> GetBalance >> %w", err)
	}

	return &account, nil
}
