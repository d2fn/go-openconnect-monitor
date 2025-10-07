package main

import (
	"os"
	"fmt"
	toml "github.com/pelletier/go-toml/v2"
)

const (
	configPath = "./config.toml"
)

type Config struct {
	Controller ControllerConfig
	DsidPoller DsidPollerConfig
	HealthCheck HealthCheckConfig
	OpenConnect OpenConnectConfig
	Vpn VPNConfig
}

type ControllerConfig struct {
	IntervalSeconds int
}

type DsidPollerConfig struct {
	CookieName string
	CookiePath string
}

type HealthCheckConfig struct {
	Host string
	Port string
	TimeoutSeconds int
}

type OpenConnectConfig struct {
	ExtraArgs string
	Verbose bool
}

type VPNConfig struct {
	Host string
	Url string
}

func LoadConfig() (Config, error) {
	config := Config{}
	tomlBytes, err := os.ReadFile(configPath)
	if err == nil {
		err = toml.Unmarshal(tomlBytes, &config)
		if err == nil {
			fmt.Printf("loaded configfile from %s\n", configPath)
			return config, nil
		}
	}
	return Config{}, err
}
