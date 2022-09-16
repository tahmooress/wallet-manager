package rabbitmq

import (
	"fmt"

	"github.com/streadway/amqp"
)

func (s *session) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)
	if err != nil {
		s.setLastHealthStatus(HealthState{
			Er:      err,
			Message: fmt.Sprintf("attempting to dial %s failed", maskConnection(s.cfg.addr)),
			Status:  false,
		})

		return nil, fmt.Errorf("rabbit >> Run() >> %w", err)
	}

	s.setConnection(conn)
	s.logger.Infoln("Connected!")

	return conn, nil
}

func (s *session) setConnection(conn *amqp.Connection) {
	s.conn = conn
	s.notifyCloseConn = make(chan *amqp.Error)
	s.conn.NotifyClose(s.notifyCloseConn)
	s.shutDown = make(chan struct{})
}

func (s *session) connectionIsClosed() bool {
	s.cnmu.RLock()
	defer s.cnmu.RUnlock()

	return !s.connected
}

func (s *session) setConnectionStatus(connected bool) {
	s.cnmu.Lock()
	defer s.cnmu.Unlock()

	s.connected = connected
}

func (s *session) isAlreadyClosed() bool {
	s.clmu.RLock()
	defer s.clmu.RUnlock()

	return s.alreadyClosed
}

func (s *session) setAlreadyClosed() {
	s.clmu.Lock()
	defer s.clmu.Unlock()

	s.alreadyClosed = true
}
