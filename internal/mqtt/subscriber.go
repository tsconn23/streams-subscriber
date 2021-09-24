package mqtt

import (
	"context"
	"encoding/json"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	logInterface "github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"github.com/project-alvarium/stream-subscriber/internal/interfaces"
	"os"
	"sync"
	"time"
)

type mqttSubscriber struct {
	chPub      chan message.SubscribeWrapper
	endpoint   config.MqttConfig
	logger     logInterface.Logger
	mqttClient MQTT.Client
}

func NewMqttSubscriber(endpoint config.MqttConfig, pub chan message.SubscribeWrapper, logger logInterface.Logger) interfaces.StreamSubscriber {
	// create MQTT options
	opts := MQTT.NewClientOptions()
	opts.AddBroker(endpoint.Provider.Uri())
	opts.SetClientID(endpoint.ClientId)
	opts.SetUsername(endpoint.User)
	opts.SetPassword(endpoint.Password)
	opts.SetCleanSession(endpoint.Cleanness)

	var subscriber = mqttSubscriber{
		chPub:      pub,
		endpoint:   endpoint,
		logger:     logger,
		mqttClient: MQTT.NewClient(opts),
	}
	// no error to report
	return &subscriber
}

func (s *mqttSubscriber) Subscribe(ctx context.Context, wg *sync.WaitGroup) bool {
	err := s.Connect()
	if err != nil {
		s.logger.Error(err.Error())
		return false
	}

	if len(s.endpoint.Topics) > 1 {
		topicsMap := make(map[string]byte)
		// build topic qos map
		for _, topic := range s.endpoint.Topics {
			topicsMap[topic] = byte(s.endpoint.Qos)
		}

		if token := s.mqttClient.SubscribeMultiple(topicsMap, s.mqttMessageHandler); token.Wait() {
			if token.Error() != nil {
				s.logger.Error(token.Error().Error())
				return false
			} else {
				s.logger.Write(logging.DebugLevel, "successfully subscribed (multiple)")
			}
		}
	} else if len(s.endpoint.Topics) == 1 {
		var topic = s.endpoint.Topics[0]
		if token := s.mqttClient.Subscribe(topic, byte(s.endpoint.Qos), s.mqttMessageHandler); token.Wait() {
			if token.Error() != nil {
				s.logger.Error(token.Error().Error())
				return false
			} else {
				s.logger.Write(logging.DebugLevel, "successfully subscribed")
			}
		}
	} else {
		s.logger.Error("at least one topic value should be configured")
		return false
	}

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		close(s.chPub)
		s.logger.Write(logging.InfoLevel, "shutdown received")
	}()
	return true
}

func (s *mqttSubscriber) Close() error {
	s.mqttClient.Disconnect(1000)
	return nil
}

func (s *mqttSubscriber) Connect() error {
	// Connect client to broker if not already connected
	if !s.mqttClient.IsConnected() {
		if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			// Connecting to publisher
			return token.Error()
		}
	}
	return nil
}

// utility function to monitor OS signals
func (s *mqttSubscriber) monitorSignals(signals chan os.Signal, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	// Looping for system interrupts then closing client
	for {
		select {
		case <-signals:
			s.logger.Write(logging.InfoLevel, "Interrupt is detected. Message Consumption stopped.")
			s.Close()
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// General message handing func
func (s *mqttSubscriber) mqttMessageHandler(client MQTT.Client, mqttMsg MQTT.Message) {
	var wrapped message.SubscribeWrapper
	err := json.Unmarshal(mqttMsg.Payload(), &wrapped)
	if err != nil {
		s.logger.Error(err.Error())
	} else {
		s.chPub <- wrapped
	}
}
