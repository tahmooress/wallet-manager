package rabbitmq

import (
	"errors"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

var errConnectionLost = errors.New("unable to publish due to connection lost or closed channel")

func (s *session) Publish(data []byte) error {
	select {
	case <-s.stopPublish:
		return errConnectionLost
	default:
		// it should refactor to not allways open a new goroutine
		// and use worker pattern to set limit for number of goroutines.
		go func() {
			s.publishChan <- data
		}()
	}

	return nil
}

func (s *session) handlePublisherConnection() {
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

		if done := s.handlePublisherChannel(conn); done {
			break
		}
	}
}

func (s *session) handlePublisherChannel(conn *amqp.Connection) bool {
	close(s.stopPublish)

	for {
		s.logger.Infoln("rabbit: initializing channel started...")

		s.setConnectionStatus(false)

		err := s.setPublisherChannelAndQueue(conn)
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
func (s *session) setPublisherChannelAndQueue(conn *amqp.Connection) error {
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

	s.updateChannel(ch)
	s.setConnectionStatus(true)
	s.stopPublish = make(chan struct{})

	go s.publisher()

	return nil
}

func (s *session) updateChannel(channel *amqp.Channel) {
	s.channel = channel
	s.notifyCloseChan = make(chan *amqp.Error)
	s.channel.NotifyClose(s.notifyCloseChan)
}

func (s *session) publisher() {
	// drain remaining item
	defer func() {
		for range s.publishChan {
		}
	}()

	for {
		select {
		case data := <-s.publishChan:
			err := s.channel.Publish(
				s.cfg.ExchangeName,
				s.cfg.RouteKey,
				false,
				false,
				amqp.Publishing{
					Headers:         amqp.Table{},
					ContentType:     "application/json",
					ContentEncoding: "",
					Body:            data,
					DeliveryMode:    amqp.Transient,
					Priority:        0,
				},
			)

			s.logger.Errorf("fail to publish error: %w", err)
		case <-s.stopPublish:
			return
		case <-s.done:
			return
		}
	}
}
