package rabbitmq

import (
	"fmt"
	"regexp"
)

func (s *session) Close() error {
	if s.isAlreadyClosed() {
		s.logger.Warningln("rabbit: trying to close rabbit despite of is already closed")

		return nil
	}

	if s.isPublisher {
		return s.closerPublisher()
	}

	return s.closeConsumer()
}

func (s *session) closeConsumer() error {
	s.logger.Infoln("rabbit: shut down started...")

	defer func() {
		s.setAlreadyClosed()
	}()

	if s.connectionIsClosed() {
		return fmt.Errorf("rabbit >> close() >> %w", errAlreadyClosed)
	}

	err := s.channel.Cancel(s.cfg.consumerTag, false)
	if err != nil {
		return fmt.Errorf("rabbit >> close() >> %w", err)
	}

	close(s.done)

	<-s.shutDown

	s.setConnectionStatus(false)

	err = s.conn.Close()
	if err != nil {
		return fmt.Errorf("rabbit >> close() >> %w", err)
	}

	close(s.shutDown)

	s.logger.Infoln("rabbit: successfully shut down")

	return nil
}

func (s *session) closerPublisher() error {
	s.logger.Infoln("rabbit: shut down started...")

	defer func() {
		s.setAlreadyClosed()
	}()

	if s.connectionIsClosed() {
		return fmt.Errorf("rabbit >> close() >> %w", errAlreadyClosed)
	}

	close(s.done)

	<-s.shutDown

	s.setConnectionStatus(false)

	err := s.conn.Close()
	if err != nil {
		return fmt.Errorf("rabbit >> close() >> %w", err)
	}

	s.channel.Close()

	close(s.shutDown)

	s.logger.Infoln("rabbit: successfully shut down")

	return nil
}

func (s *session) CheckHealth() HealthState {
	return s.getLastHealthStatus()
}

func (s *session) getLastHealthStatus() HealthState {
	s.lstmu.RLock()
	defer s.lstmu.RUnlock()

	return *s.lastStatus
}

func (s *session) setLastHealthStatus(h HealthState) {
	s.lstmu.Lock()
	defer s.lstmu.Unlock()

	s.lastStatus = &h
}

func maskConnection(connString string) string {
	reg := regexp.MustCompile(`\/\/.*:.*@`)

	return reg.ReplaceAllString(connString, "${1}//*:*")
}
