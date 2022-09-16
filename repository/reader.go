package repository

import (
	"context"
	"fmt"

	"github.com/tahmooress/wallet-manager/entities"
)

func (r *repo) GetBalance(ctx context.Context, user string) (*entities.Account, error) {
	query := `SELECT * FROM accounts WHERE user = $1`

	var account entities.Account

	err := r.db.QueryRowContext(ctx, query, user).Scan(
		&account.User, &account.Balance, &account.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository >> GetBalance >> %w", err)
	}

	return &account, nil
}
