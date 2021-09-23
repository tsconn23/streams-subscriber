package config

import (
	"encoding/json"
	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/provider-logging/pkg/config"
)

// ApplicationConfig serves as the root node for configuration and contains targeted child types with specialized
// concerns.
type ApplicationConfig struct {
	Stream  	SdkConfig.StreamInfo	`json:"stream,omitempty"`
	Logging 	config.LoggingInfo		`json:"logging,omitempty"`
}

func (a ApplicationConfig) AsString() string {
	b, _ := json.Marshal(a)
	return string(b)
}