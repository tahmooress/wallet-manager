package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tahmooress/wallet-manager/logger/hooks"
)

var (
	errEmptySentryDNS     = errors.New("dns in sentry config must be not empty")
	errSentryFlushTimeout = errors.New("sentry flush timeout")
)

type logger struct {
	entry           *logrus.Entry
	file            io.Closer
	hooks           []logrus.Hook
	sentryNeedFlush bool
}

func New(cfg Config) (Logger, error) {
	lg := new(logger)

	lg.entry = &logrus.Entry{
		Logger: &logrus.Logger{
			Out:       os.Stdout,
			Hooks:     make(logrus.LevelHooks),
			Formatter: getFormmater(cfg.Formmater),
			Level:     getlogLevel(cfg.LogLevel),
		},
	}

	lg.setHostnameField()

	err := lg.setLoggingFile(cfg.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("logger: New() >> %w", err)
	}

	defer lg.setHooks()
	// in case of error opened file should close.
	defer func() {
		if err != nil {
			lg.file.Close()
		}
	}()

	if cfg.Sentry != nil {
		if err = lg.sentryHook(cfg.Sentry); err != nil {
			return nil, fmt.Errorf("logger: New() >> %w", err)
		}
	}

	if cfg.APM != nil {
		if err = lg.apmHook(cfg.APM); err != nil {
			return nil, fmt.Errorf("logger: New() >> %w", err)
		}
	}

	return lg, nil
}

func (l *logger) setLoggingFile(filePath string) error {
	if filePath == "" {
		l.entry.Logger.Out = os.Stdout
		l.file = os.Stdout

		return nil
	}

	var file *os.File

	_, err := os.Stat(filePath)

	flags := os.O_RDWR | os.O_APPEND

	if os.IsNotExist(err) {
		flags |= os.O_CREATE
		dirPath := path.Dir(filePath)

		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("logger: setLoggingFile >> %w", err)
		}
	}

	file, err = os.OpenFile(filePath, flags, os.ModePerm)
	if err != nil {
		return fmt.Errorf("logger: setLoggingFile >> %w", err)
	}

	multiOut := io.MultiWriter(file, os.Stdout)

	l.file = file
	l.entry.Logger.Out = multiOut

	return nil
}

func (l *logger) sentryHook(s *Sentry) error {
	if s == nil {
		return nil
	}

	const (
		defaultEnvironment = "debug"
		defaultSampleRate  = 0.5
	)

	if s.DNS == "" {
		return errEmptySentryDNS
	}

	if s.Environment == "" {
		s.Environment = defaultEnvironment
	}

	if s.SampleRate == 0 {
		s.SampleRate = defaultSampleRate
	}

	levels := getLevels(s.LogLevels)

	hook, err := hooks.NewSentryHook(s.DNS, s.Environment, s.SampleRate, levels)
	if err != nil {
		return fmt.Errorf("sentryHook >> %w", err)
	}

	l.hooks = append(l.hooks, hook)
	l.sentryNeedFlush = true

	return nil
}

func (l *logger) apmHook(a *APM) error {
	return nil
}

func (l *logger) setHooks() {
	for _, h := range l.hooks {
		l.entry.Logger.AddHook(h)
	}
}

func (l *logger) setHostnameField() {
	hostName, _ := os.Hostname()

	l.entry = l.entry.Logger.WithField("@Host.Name", hostName)
}

func getFormmater(f *Formmater) logrus.Formatter {
	if f == nil || (f.FormatType == "" && f.CustomFormmater == nil) {
		return defaultTextFormmater
	}

	switch f.FormatType {
	case TextFormat:
		return defaultTextFormmater
	case JSNOFormat:
		return defaultJSONFormmater
	}

	return f.CustomFormmater
}

func getlogLevel(level string) logrus.Level {
	logLevel := map[string]logrus.Level{
		"panic": logrus.PanicLevel,
		"fatal": logrus.FatalLevel,
		"error": logrus.ErrorLevel,
		"warn":  logrus.WarnLevel,
		"info":  logrus.InfoLevel,
		"debug": logrus.DebugLevel,
		"trace": logrus.TraceLevel,
	}

	logrusLevel, ok := logLevel[strings.ToLower(level)]
	if !ok {
		return logLevel["info"]
	}

	return logrusLevel
}

func getLevels(levels []Level) []logrus.Level {
	if len(levels) == 0 {
		return []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		}
	}

	lgLevels := make([]logrus.Level, len(levels))

	for i := range levels {
		lgLevels[i] = logrus.Level(levels[i])
	}

	return lgLevels
}
