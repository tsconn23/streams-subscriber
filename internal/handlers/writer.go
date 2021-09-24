package handlers

import (
	"context"
	"encoding/json"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	logInterface "github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"sync"
)

type ConsoleWriter struct {
	chMsg    chan message.SubscribeWrapper
	logger   logInterface.Logger
}

func NewConsoleWriter(chRecv chan message.SubscribeWrapper, logger logInterface.Logger) ConsoleWriter {
	w := ConsoleWriter{
		chMsg:  chRecv,
		logger: logger,
	}
	return w
}

func (w *ConsoleWriter) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, ok := <-w.chMsg
			if ok {
				b, _ := json.Marshal(msg)
				w.logger.Write(logging.DebugLevel, string(b))
			} else {
				return
			}
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		w.logger.Write(logging.InfoLevel, "shutdown received")
	}()
	return true
}
