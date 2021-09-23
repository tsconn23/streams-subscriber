package handlers

import (
	"context"
	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	logInterface "github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"github.com/project-alvarium/stream-subscriber/internal/config"
	"github.com/project-alvarium/stream-subscriber/internal/interfaces"
	"github.com/project-alvarium/stream-subscriber/internal/iota"
	"sync"
	"time"
)

type Subscription struct {
	cfg		config.ApplicationConfig
	logger  logInterface.Logger
}

func NewSubscription(cfg config.ApplicationConfig, logger logInterface.Logger) Subscription {
	return Subscription{
		cfg: cfg,
		logger: logger,
	}
}

func (s *Subscription) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	var subscriber interfaces.StreamSubscriber
	if s.cfg.Stream.Type == contracts.IotaStream {
		info, ok := s.cfg.Stream.Config.(SdkConfig.IotaStreamConfig)
		if !ok {
			s.logger.Error("invalid cast for IotaStream")
			return false
		}
		subscriber = iota.NewIotaSubscriber(info, s.logger)
	}

	err := subscriber.Connect()
	if err != nil {
		s.logger.Error(err.Error())
		return false
	}

	cancelled := false
	wg.Add(1)
	go func() {
		defer wg.Done()

		for !cancelled {
			time.Sleep(100 * time.Millisecond)
			err := subscriber.Read()
			if err != nil {
				s.logger.Error(err.Error())
			}
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		s.logger.Write(logging.InfoLevel, "shutdown received")
		cancelled = true
	}()
	return true
}