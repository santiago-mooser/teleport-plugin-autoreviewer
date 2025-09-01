package config

import (
	"time"
)

// Config defines the configuration for the teleport-autoreviewer service.
type Config struct {
	Teleport struct {
		Addr                    string        `yaml:"addr"`
		Identity                string        `yaml:"identity"`
		Reviewer                string        `yaml:"reviewer"`
		IdentityRefreshInterval time.Duration `yaml:"identity_refresh_interval"`
	} `yaml:"teleport"`

	Server struct {
		HealthPort int    `yaml:"health_port"`
		HealthPath string `yaml:"health_path"`
	} `yaml:"server"`

	Rejection struct {
		DefaultMessage string          `yaml:"default_message"`
		Rules          []RejectionRule `yaml:"rules"`
	} `yaml:"rejection"`
}

// RejectionRule defines a single rejection rule with regex pattern and custom message.
type RejectionRule struct {
	Name        string `yaml:"name"`
	ReasonRegex string `yaml:"reason_regex"`
	Message     string `yaml:"message"`
	RolesRegex  string `yaml:"roles_regex,omitempty"`
}
