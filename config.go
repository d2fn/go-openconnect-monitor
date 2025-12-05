package main

import (
	"fmt"
	toml "github.com/pelletier/go-toml/v2"
	"os"
)

type Config struct {
	Controller  ControllerConfig
	DsidWriter  DsidWriterConfig
	DsidCookiePoller  DsidCookiePollerConfig
	HealthCheck HealthCheckConfig
	OpenConnect OpenConnectConfig
	Vpn         VPNConfig
}

type ControllerConfig struct {
	IntervalSeconds               int
	HealthCheckGracePeriodSeconds int
}

type DsidCookiePollerConfig struct {
	CookieName string
	CookiePath string
	CookieHost string
}

type DsidWriterConfig struct {
	IntervalSeconds int
}

type HealthCheckConfig struct {
	Host           string
	Port           string
	TimeoutSeconds int
}

type OpenConnectConfig struct {
	ExtraArgs                  string
	Verbose                    bool
	DryRun                     bool
	ShutdownGracePeriodSeconds int
}

type VPNConfig struct {
	Url string
}

func LoadConfig(configPath string) (Config, error) {
	config := Config{}
	tomlBytes, err := os.ReadFile(configPath)
	if err == nil {
		err = toml.Unmarshal(tomlBytes, &config)
		if err == nil {
			fmt.Printf("loaded config file from %s\n", configPath)
			return config, nil
		}
	}
	return Config{}, err
}
