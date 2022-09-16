package service

import (
	"encoding/json"

	"github.com/streadway/amqp"
	"github.com/tahmooress/wallet-manager/broker/rabbitmq"
	"github.com/tahmooress/wallet-manager/configs"
	"github.com/tahmooress/wallet-manager/entities"
	"github.com/tahmooress/wallet-manager/logger"
	"github.com/tahmooress/wallet-manager/pkg/wrapper"
	"github.com/tahmooress/wallet-manager/repository"
)

type service struct {
	db repository.DB

	consumer rabbitmq.Consumer

	closer *wrapper.Closer
	logger logger.Logger
}

func New(cfg *configs.AppConfigs, logger logger.Logger) (u Usecases, err error) {
	c := new(wrapper.Closer)

	db, err := repository.New(cfg)
	if err != nil {
		return nil, err
	}

	c.Add(db)

	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	s := &service{
		db:     db,
		closer: c,
		logger: logger,
	}

	err = s.initConsumer(cfg)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// register handler to consumer and start consuming.
func (s *service) initConsumer(cfg *configs.AppConfigs) error {
	handler := func(d amqp.Delivery) (action rabbitmq.Action) {
		var voucher entities.Voucher

		err := json.Unmarshal(d.Body, &voucher)
		if err != nil {
			return rabbitmq.NackDiscard
		}

		err = s.AddBalance(&voucher)
		if err != nil {
			return rabbitmq.NackRequeue
		}

		return rabbitmq.Ack
	}

	consumer, err := rabbitmq.NewConsumer(
		rabbitmq.Config{
			Host:         cfg.RabbitMQWalletHost,
			Port:         cfg.RabbitMQWalletPort,
			ExchangeName: cfg.RabbitMQWalletExchange,
			ExchangeType: cfg.RabbitMQWalletExchangeType,
			RouteKey:     cfg.RabbitMQWalletRoutingKey,
			Queue:        cfg.RabbitMQWalletQuee,
		},
		handler,
		s.logger,
	)
	if err != nil {
		return err
	}

	s.consumer = consumer
	s.closer.Add(consumer)

	return nil
}

func (s *service) Close() error {
	return s.closer.Close()
}
