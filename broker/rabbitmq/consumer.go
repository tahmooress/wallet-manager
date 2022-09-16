package rabbitmq

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

func (s *session) handleConsumerConnection(handler Handler) {
	for {
		s.setConnectionStatus(false)

		s.logger.Infoln("Attempting to connect")

		conn, err := s.connect(s.cfg.addr)
		if err != nil {
			s.logger.Warningln("Failed to connect. Retrying...")

			select {
			case <-s.done:
				return
			case <-time.After(s.cfg.reconnectInterval):
			}

			continue
		}

		if done := s.handleConsumerChannel(conn, handler); done {
			break
		}
	}
}

func (s *session) handleConsumerChannel(conn *amqp.Connection, handler Handler) bool {
	for {
		s.logger.Infoln("rabbit: initializing channel started...")

		s.setConnectionStatus(false)

		err := s.setConsumerChannelAndQueue(conn, handler)
		if err != nil {
			s.logger.Infoln("Failed to initialize channel. Retrying...")

			select {
			case <-s.done:
				return true
			case <-time.After(s.cfg.reInitInterval):
			}

			continue
		}

		s.setLastHealthStatus(HealthState{
			Er:     nil,
			Status: true,
			Message: fmt.Sprintf("RabbitMQ Consumer is up on: %s for queue: %s, routingKey: %s"+
				"exchangeType: %s, exchangeName: %s", maskConnection(s.cfg.addr), s.cfg.Queue,
				s.cfg.RouteKey, s.cfg.ExchangeType, s.cfg.ExchangeName),
		})

		select {
		case <-s.done:
			s.setLastHealthStatus(HealthState{
				Er:     nil,
				Status: false,
				Message: fmt.Sprintf("context cancel Called for connection: %s"+
					"should restart the app", maskConnection(s.cfg.addr)),
			})

			return true
		case err := <-s.notifyCloseConn:
			s.logger.Infoln("Connection closed. Reconnecting...")

			s.setLastHealthStatus(HealthState{
				Er:     err,
				Status: false,
				Message: fmt.Sprintf("connect with address %s closed, Reconnection started...",
					maskConnection(s.cfg.addr)),
			})

			return false
		case err := <-s.notifyCloseChan:
			s.logger.Infoln("Channel closed. Re-running init...")

			s.setLastHealthStatus(HealthState{
				Er:     err,
				Status: false,
				Message: fmt.Sprintf("channel for queue: %s with routingKey: %s closed"+
					"Re-running init started...", s.cfg.Queue, s.cfg.RouteKey),
			})
		}
	}
}

// nolint : funlen
func (s *session) setConsumerChannelAndQueue(conn *amqp.Connection, handler Handler) error {
	ch, err := conn.Channel()
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:     err,
			Status: false,
			Message: fmt.Sprintf("cant open channel on connection for %s",
				maskConnection(s.cfg.addr)),
		})

		return fmt.Errorf("rabbit >> setupChannel() >> %w", err)
	}

	err = ch.ExchangeDeclare(s.cfg.ExchangeName, s.cfg.ExchangeType, true,
		false, false, false, nil)
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:     err,
			Status: false,
			Message: fmt.Sprintf("cant declare exchange on connection for %s, exchangeName: %s, exchangeType: %s",
				maskConnection(s.cfg.addr), s.cfg.ExchangeName, s.cfg.ExchangeType),
		})

		return fmt.Errorf("rabbit >> setupChannelAndQueue() >> %w", err)
	}

	_, err = ch.QueueDeclare(s.cfg.Queue, true, false, false, false, nil)
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:     err,
			Status: false,
			Message: fmt.Sprintf("cant declare queue for connection: %s, queueName: %s",
				maskConnection(s.cfg.addr), s.cfg.Queue),
		})

		return fmt.Errorf("rabbit >> setupChannelAndQueue() >> %w", err)
	}

	err = ch.Qos(int(s.cfg.prefetchCount), int(s.cfg.prefetchSize), false)
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:     err,
			Status: false,
			Message: fmt.Sprintf("cant setup prefetchSize and prefetchCount for connection: %s,"+
				"prefetchSize: %d, prefetchCount: %d",
				maskConnection(s.cfg.addr), s.cfg.prefetchSize, s.cfg.prefetchCount),
		})

		return fmt.Errorf("rabbit >> setupChannelAndQueue() >> %w", err)
	}

	err = ch.QueueBind(s.cfg.Queue, s.cfg.RouteKey, s.cfg.ExchangeName, false, nil)
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:     err,
			Status: false,
			Message: fmt.Sprintf("cant bind routing key of queue to exchange, queueName: %s,"+
				" routingKey: %s, exchangeName: %s", s.cfg.Queue, s.cfg.RouteKey, s.cfg.ExchangeName),
		})

		return fmt.Errorf("rabbit >> setupChannelAndQueue() >> %w", err)
	}

	consumer, err := ch.Consume(s.cfg.Queue, s.cfg.consumerTag, false, false, false, false, nil)
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:     err,
			Status: false,
			Message: fmt.Sprintf("fail to start consuming for queueName: %s, with consumerTag: %s",
				s.cfg.Queue, s.cfg.consumerTag),
		})

		return fmt.Errorf("rabbit >> setupChannelAndQueue() >> %w", err)
	}

	s.updateChannelAndConsumer(ch, consumer)
	s.setConnectionStatus(true)

	go s.worker(handler)

	return nil
}

func (s *session) updateChannelAndConsumer(channel *amqp.Channel, consumer <-chan amqp.Delivery) {
	s.channel = channel
	s.notifyCloseChan = make(chan *amqp.Error)
	s.channel.NotifyClose(s.notifyCloseChan)
	s.consumer = consumer
}

func (s *session) worker(handler Handler) {
	for {
		select {
		case delivery := <-s.consumer:
			s.handlerCaller(delivery, handler)
		case <-s.done:
			for len(s.consumer) > 0 {
				s.handlerCaller(<-s.consumer, handler)
			}

			s.shutDown <- struct{}{}

			return
		}
	}
}

func (s *session) handlerCaller(d amqp.Delivery, handler Handler) {
	switch handler(d) {
	case Ack:
		err := d.Ack(false)
		if err != nil {
			s.logger.Errorf("can't ack message: %v", err)
		}
	case NackDiscard:
		err := d.Nack(false, false)
		if err != nil {
			s.logger.Errorf("can't nack message: %v", err)
		}
	case NackRequeue:
		err := d.Nack(false, true)
		if err != nil {
			s.logger.Errorf("can't nack message: %v", err)
		}
	}
}
