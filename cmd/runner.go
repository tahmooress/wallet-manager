package cmd

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/tahmooress/wallet-manager/api"
	"github.com/tahmooress/wallet-manager/configs"
	"github.com/tahmooress/wallet-manager/logger"
	"github.com/tahmooress/wallet-manager/pkg/wrapper"
	"github.com/tahmooress/wallet-manager/service"
)

func Runner() (closer io.Closer, errChan <-chan error, err error) {
	cfg := configs.Load()

	log, err := logger.New(logger.Config{
		LogFilePath: cfg.LogFilePath,
		LogLevel:    cfg.LogLevel,
	})
	if err != nil {
		return nil, nil, err
	}

	c := new(wrapper.Closer)

	c.Add(log)

	defer func() {
		if err != nil {
			_ = c.Close()
		}
	}()

	srv, err := service.New(cfg, log)
	if err != nil {
		return nil, nil, err
	}

	c.Add(srv)

	server, errChan, err := api.NewHTTPServer(cfg, nil, log)
	if err != nil {
		return nil, nil, err
	}

	c.Add(server)

	return c, nil, nil
}

func Shutdown(errChan <-chan error, closer io.Closer) int {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	defer close(c)

	var existStatus int

	select {
	case <-c:
		err := closer.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "terminate signal received -> closer error: %s\n", err)

			existStatus = 1
		} else {
			fmt.Fprintf(os.Stdout, "terminate signal received -> shutdowned cleanly")
		}

		break
	case err := <-errChan:
		fmt.Fprintln(os.Stderr, err)

		break
	}

	return existStatus
}
