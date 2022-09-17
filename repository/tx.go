package repository

import (
	"database/sql"
	"fmt"

	"github.com/tahmooress/wallet-manager/entities"
)

type transaction struct {
	tx *sql.Tx
}

func (t *transaction) AddBalance(voucher *entities.Voucher) error {
	query := `INSERT INTO accounts(mobile, balance)
	VALUES($1,$2) ON CONFLICT
	 DO UPDATE SET balance = accounts.balance + $2`

	_, err := t.tx.Exec(query, voucher.Mobile, voucher.Value)
	if err != nil {
		return fmt.Errorf("repository >> AddBalance >> %w", err)
	}

	return nil
}
