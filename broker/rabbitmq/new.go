package rabbitmq

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/tahmooress/wallet-manager/logger"
)

const (
	defaultReconnectInterval = 3 * time.Second
	defaultReInitInterval    = 1 * time.Second
	defaultPrefetchCount     = 1
	defaultPrefetchSize      = 0
)

var (
	errEmptyValue    = errors.New("some variable in configs are empty")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errEmptyHandler  = errors.New("handler should not be empty")
)

// Action is an action that occurs after processed this delivery.
type Action int

// Handler defines the handler of each Delivery and return Action.
type Handler func(d amqp.Delivery) (action Action)

const (
	// Ack default ack this msg after you have successfully processed this delivery.
	Ack Action = iota
	// NackDiscard the message will be dropped or delivered to a server configured dead-letter queue.
	NackDiscard
	// NackRequeue deliver this message to a different consumer.
	NackRequeue
)

type Consumer interface {
	CheckHealth() HealthState
	Close() error
}

type Publisher interface {
	Publish(data []byte) error
	CheckHealth() HealthState
	Close() error
}

type session struct {
	cfg         Config
	isPublisher bool

	conn            *amqp.Connection
	channel         *amqp.Channel
	notifyCloseConn chan *amqp.Error
	notifyCloseChan chan *amqp.Error

	consumer <-chan amqp.Delivery

	publishChan chan []byte
	stopPublish chan struct{}

	done     chan bool
	shutDown chan struct{}

	cnmu      *sync.RWMutex
	connected bool

	clmu          *sync.RWMutex
	alreadyClosed bool

	lstmu      *sync.RWMutex
	lastStatus *HealthState

	logger logger.Logger
}

func NewConsumer(cfg Config, handler Handler, logger logger.Logger) (Consumer, error) {
	if handler == nil {
		return nil, errEmptyHandler
	}

	session := session{
		isPublisher: false,
		done:        make(chan bool),
		cnmu:        &sync.RWMutex{},
		clmu:        &sync.RWMutex{},
		lstmu:       &sync.RWMutex{},
		logger:      logger,
	}

	err := session.initSetting(cfg)
	if err != nil {
		return nil, fmt.Errorf("rabbit >> run() >> %w", err)
	}

	session.lastStatus = &HealthState{
		Status: false,
		Message: fmt.Sprintf("connection on %s is open but Start Method not called",
			maskConnection(cfg.addr)),
	}

	go session.handleConsumerConnection(handler)

	return &session, nil
}

func NewPublisher(cfg Config, logger logger.Logger) (Publisher, error) {
	session := session{
		isPublisher: true,
		done:        make(chan bool),
		publishChan: make(chan []byte),
		stopPublish: make(chan struct{}),
		cnmu:        &sync.RWMutex{},
		clmu:        &sync.RWMutex{},
		lstmu:       &sync.RWMutex{},
		logger:      logger,
	}

	err := session.initSetting(cfg)
	if err != nil {
		return nil, fmt.Errorf("rabbit >> run() >> %w", err)
	}

	session.lastStatus = &HealthState{
		Status: false,
		Message: fmt.Sprintf("connection on %s is open but Start Method not called",
			maskConnection(cfg.addr)),
	}

	go session.handlePublisherConnection()

	return &session, nil
}

func (s *session) validate(cfg Config) error {
	if cfg.Host == "" || cfg.Port == "" ||
		cfg.ExchangeName == "" || cfg.ExchangeType == "" ||
		cfg.Queue == "" || cfg.RouteKey == "" {
		return errEmptyValue
	}

	return nil
}

// nolint : cyclop
func (s *session) initSetting(cfg Config) error {
	err := s.validate(cfg)
	if err != nil {
		return fmt.Errorf("rabbit >> initSetting >> %w", err)
	}

	cfg.addr = fmt.Sprintf("%s://%s", "amqp", cfg.Host)

	min := 10000
	max := 99999

	rand.Seed(time.Now().UnixNano())
	tagRand := min + rand.Intn(max-min+1) //nolint: gosec
	s.cfg.consumerTag = fmt.Sprintf("go-webhook_%d", tagRand)

	s.cfg = cfg

	s.cfg.reconnectInterval = defaultReconnectInterval
	s.cfg.reInitInterval = defaultReInitInterval

	s.cfg.prefetchCount = defaultPrefetchCount

	s.cfg.prefetchSize = defaultPrefetchSize

	return nil
}
