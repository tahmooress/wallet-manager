package repository

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/tahmooress/wallet-manager/configs"
)

type repo struct {
	db *sql.DB
}

func New(config *configs.AppConfigs) (DB, error) {
	DatabaseDriver := config.DatabaseDriver
	DatabaseName := config.DatabaseName
	DatabaseHost := config.DatabaseHost
	DatabasePort := config.DatabasePort
	DatabaseUser := config.DatabaseUser
	DatabasePass := config.DatabasePassword

	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DatabaseHost, DatabasePort, DatabaseUser, DatabasePass, DatabaseName)

	db, err := sql.Open(DatabaseDriver, dbinfo)
	if err != nil {
		return nil, fmt.Errorf("repository initialize , Establishing a Database Connection Failed >> %w", err)
	}

	if err = db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("repository initialize , Database Connection Test Failed >> %w", err)
	}

	return &repo{
		db: db,
	}, nil
}

func (r *repo) begin() (Tx, error) {
	tx, err := r.db.BeginTx(context.TODO(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: Begin >> %w", err)
	}

	return &transaction{
		tx: tx,
	}, nil
}

func (r *repo) ExecWrite(fn func(tx Tx) error) error {
	tx, err := r.begin()
	if err != nil {
		return fmt.Errorf("repository: ExecWrite >> %w", err)
	}

	err = fn(tx)
	if err != nil {
		_ = tx.(*transaction).tx.Rollback()

		return err
	}

	return tx.(*transaction).tx.Commit()
}

func (r *repo) Close() error {
	return r.db.Close()
}
