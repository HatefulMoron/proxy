package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"proxy/config"
	"proxy/logging"
	"proxy/proxy"
)

func TestProxyServiceIntegration(t *testing.T) {
	mockTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test-Header", "target-response")

		response := map[string]interface{}{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.RawQuery,
			"headers": r.Header,
		}

		if r.Method == "POST" && r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			response["body"] = string(body)
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer mockTarget.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 0,
			Host: "127.0.0.1",
		},
		Target: config.TargetConfig{
			Scheme: "http",
			Host:   mockTarget.URL[7:],
		},
		Proxy: config.ProxyConfig{
			URL:      "http://127.0.0.1:8888",
			Username: "test-user",
			Password: "test-pass",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	var logBuffer bytes.Buffer
	logger := logging.NewLogger(&cfg.Logging)
	logger.SetOutput(&logBuffer)

	cfg.Proxy.URL = "invalid-url-for-test"
	handler, _ := proxy.NewHandler(cfg, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/metrics", handler.Metrics)
	mux.HandleFunc("/", handler.ServeHTTP)

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("Health check endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", resp.StatusCode)
		}
	})

	t.Run("Metrics endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/metrics")
		if err != nil {
			t.Fatalf("Metrics request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected content-type application/json, got %s", resp.Header.Get("Content-Type"))
		}
	})

	t.Run("Error handling for invalid proxy", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/test")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadGateway {
			t.Errorf("Expected status 502, got %d", resp.StatusCode)
		}

		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected content-type application/json, got %s", resp.Header.Get("Content-Type"))
		}
	})
}

func TestConfigValidationInHandler(t *testing.T) {
	t.Run("Empty proxy URL should cause error", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 8080,
				Host: "127.0.0.1",
			},
			Target: config.TargetConfig{
				Scheme: "https",
				Host:   "example.com",
			},
			Proxy: config.ProxyConfig{
				URL:      "",
				Username: "test-user",
				Password: "test-pass",
			},
			Logging: config.LoggingConfig{
				Level:  "info",
				Format: "json",
			},
		}

		_, err := proxy.NewHandler(cfg, logging.NewLogger(&cfg.Logging))
		if err == nil {
			t.Errorf("Expected error for empty proxy URL")
		}
	})

	t.Run("Invalid proxy URL should cause error", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 8080,
				Host: "127.0.0.1",
			},
			Target: config.TargetConfig{
				Scheme: "https",
				Host:   "example.com",
			},
			Proxy: config.ProxyConfig{
				URL:      "://invalid",
				Username: "test-user",
				Password: "test-pass",
			},
			Logging: config.LoggingConfig{
				Level:  "info",
				Format: "json",
			},
		}

		_, err := proxy.NewHandler(cfg, logging.NewLogger(&cfg.Logging))
		if err == nil {
			t.Errorf("Expected error for invalid proxy URL")
		}
	})
}