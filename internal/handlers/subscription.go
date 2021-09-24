package handlers

import (
	"errors"
	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	logInterface "github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/stream-subscriber/internal/config"
	"github.com/project-alvarium/stream-subscriber/internal/interfaces"
	"github.com/project-alvarium/stream-subscriber/internal/iota"
	"github.com/project-alvarium/stream-subscriber/internal/mqtt"
)

// Subscription is essentially the factory for the different supported streaming platforms.
type Subscription struct {
	cfg		config.ApplicationConfig
	logger  logInterface.Logger
}

func NewSubscription(cfg config.ApplicationConfig, pub chan message.SubscribeWrapper, logger logInterface.Logger) (interfaces.StreamSubscriber, error) {
	var subscriber interfaces.StreamSubscriber
	switch cfg.Stream.Type{
	case contracts.IotaStream:
		info, ok := cfg.Stream.Config.(SdkConfig.IotaStreamConfig)
		if !ok {
			return nil, errors.New("invalid cast for IotaStream")
		}
		subscriber = iota.NewIotaSubscriber(info, pub, logger)
	case contracts.MqttStream:
		info, ok := cfg.Stream.Config.(SdkConfig.MqttConfig)
		if !ok {
			return nil, errors.New("invalid cast for MqttStream")
		}
		subscriber = mqtt.NewMqttSubscriber(info, pub, logger)
	default:
		return nil, errors.New("invalid value supplied for StreamType")
	}

	return subscriber, nil
}
