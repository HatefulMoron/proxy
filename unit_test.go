package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"proxy/config"
	"proxy/logging"
)

func TestConfigValidation(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		_, err := config.LoadConfig("nonexistent")
		if err == nil {
			t.Errorf("Expected error for nonexistent config file")
		}
	})

	t.Run("Default configuration", func(t *testing.T) {
		cfg := config.DefaultConfig()
		if cfg.Server.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
		}
		if cfg.Target.Host != "target.com" {
			t.Errorf("Expected default target host 'target.com', got %s", cfg.Target.Host)
		}
		if cfg.Proxy.URL != "http://dc.oxylabs.io:8000" {
			t.Errorf("Expected default proxy URL, got %s", cfg.Proxy.URL)
		}
	})

	t.Run("Environment variable override", func(t *testing.T) {
		os.Setenv("PROXY_PORT", "9999")
		os.Setenv("TARGET_HOST", "example.com")
		defer func() {
			os.Unsetenv("PROXY_PORT")
			os.Unsetenv("TARGET_HOST")
		}()

		cfg := config.DefaultConfig()

		file, err := os.CreateTemp("", "config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(file.Name())

		encoder := json.NewEncoder(file)
		if err := encoder.Encode(cfg); err != nil {
			t.Fatalf("Failed to encode config: %v", err)
		}
		file.Close()

		loadedCfg, err := config.LoadConfig(file.Name())
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.Server.Port != 9999 {
			t.Errorf("Expected port 9999 from env var, got %d", loadedCfg.Server.Port)
		}

		if loadedCfg.Target.Host != "example.com" {
			t.Errorf("Expected host example.com from env var, got %s", loadedCfg.Target.Host)
		}
	})
}

func TestLoggingFunctionality(t *testing.T) {
	t.Run("JSON logging format", func(t *testing.T) {
		var buffer bytes.Buffer
		cfg := &config.LoggingConfig{
			Level:  "info",
			Format: "json",
		}
		logger := logging.NewLogger(cfg)
		logger.SetOutput(&buffer)

		logger.Info("test message", map[string]interface{}{
			"key": "value",
			"number": 42,
		})

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buffer.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		if logEntry["message"] != "test message" {
			t.Errorf("Expected message 'test message', got %v", logEntry["message"])
		}

		if logEntry["level"] != "INFO" {
			t.Errorf("Expected level 'INFO', got %v", logEntry["level"])
		}

		fields, ok := logEntry["fields"].(map[string]interface{})
		if !ok {
			t.Fatalf("Fields not found or not a map")
		}

		if fields["key"] != "value" {
			t.Errorf("Expected field key=value, got %v", fields["key"])
		}

		if fields["number"].(float64) != 42 {
			t.Errorf("Expected field number=42, got %v", fields["number"])
		}
	})

	t.Run("Text logging format", func(t *testing.T) {
		var buffer bytes.Buffer
		cfg := &config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		}
		logger := logging.NewLogger(cfg)
		logger.SetOutput(&buffer)

		logger.Debug("debug message", map[string]interface{}{
			"component": "test",
		})

		output := buffer.String()
		if !bytes.Contains([]byte(output), []byte("debug message")) {
			t.Errorf("Expected output to contain 'debug message', got: %s", output)
		}

		if !bytes.Contains([]byte(output), []byte("DEBUG")) {
			t.Errorf("Expected output to contain 'DEBUG', got: %s", output)
		}

		if !bytes.Contains([]byte(output), []byte("component=test")) {
			t.Errorf("Expected output to contain 'component=test', got: %s", output)
		}
	})

	t.Run("Log level filtering", func(t *testing.T) {
		var buffer bytes.Buffer
		cfg := &config.LoggingConfig{
			Level:  "error",
			Format: "json",
		}
		logger := logging.NewLogger(cfg)
		logger.SetOutput(&buffer)

		logger.Debug("debug message", nil)
		logger.Info("info message", nil)
		logger.Warn("warn message", nil)
		logger.Error("error message", nil)

		lines := bytes.Split(buffer.Bytes(), []byte("\n"))
		nonEmptyLines := 0
		for _, line := range lines {
			if len(bytes.TrimSpace(line)) > 0 {
				nonEmptyLines++
			}
		}

		if nonEmptyLines != 1 {
			t.Errorf("Expected 1 log entry (only ERROR), got %d. Output: %s", nonEmptyLines, buffer.String())
		}
	})
}

func TestConfigurationFileCreation(t *testing.T) {
	t.Run("Create default config when missing", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := tempDir + "/config.json"

		_, err := loadConfigWithFallback(configPath)
		if err != nil {
			t.Fatalf("Failed to load config with fallback: %v", err)
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", configPath)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("Failed to load created config: %v", err)
		}

		if cfg.Server.Port != 8080 {
			t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
		}
	})
}