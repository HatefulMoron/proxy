package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server ServerConfig `json:"server"`
	Target TargetConfig `json:"target"`
	Proxy  ProxyConfig  `json:"proxy"`
	Logging LoggingConfig `json:"logging"`
}

type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

type TargetConfig struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
}

type ProxyConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	applyEnvironmentOverrides(&config)

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func applyEnvironmentOverrides(config *Config) {
	if port := os.Getenv("PROXY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	if host := os.Getenv("PROXY_HOST"); host != "" {
		config.Server.Host = host
	}

	if targetHost := os.Getenv("TARGET_HOST"); targetHost != "" {
		config.Target.Host = targetHost
	}

	if targetScheme := os.Getenv("TARGET_SCHEME"); targetScheme != "" {
		config.Target.Scheme = targetScheme
	}

	if proxyURL := os.Getenv("PROXY_URL"); proxyURL != "" {
		config.Proxy.URL = proxyURL
	}

	if proxyUser := os.Getenv("PROXY_USERNAME"); proxyUser != "" {
		config.Proxy.Username = proxyUser
	}

	if proxyPass := os.Getenv("PROXY_PASSWORD"); proxyPass != "" {
		config.Proxy.Password = proxyPass
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
}

func validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Target.Host == "" {
		return fmt.Errorf("target host is required")
	}

	if config.Target.Scheme != "http" && config.Target.Scheme != "https" {
		return fmt.Errorf("invalid target scheme: %s", config.Target.Scheme)
	}

	if config.Proxy.URL == "" {
		return fmt.Errorf("proxy URL is required")
	}

	if config.Proxy.Username == "" {
		return fmt.Errorf("proxy username is required")
	}

	if config.Proxy.Password == "" {
		return fmt.Errorf("proxy password is required")
	}

	return nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Target: TargetConfig{
			Scheme: "https",
			Host:   "target.com",
		},
		Proxy: ProxyConfig{
			URL:      "http://dc.oxylabs.io:8000",
			Username: "user-ecosuite_p5OUw-country-US",
			Password: "xFV5WETRRky_",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}