package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proxy/config"
	"proxy/logging"
	"proxy/proxy"
)

func main() {
	configPath := flag.String("config", "config/config.json", "Path to configuration file")
	flag.Parse()

	cfg, err := loadConfigWithFallback(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logger := logging.NewLogger(&cfg.Logging)
	logger.Info("Starting proxy server", map[string]interface{}{
		"version": "1.0.0",
		"config":  *configPath,
		"port":    cfg.Server.Port,
	})

	handler, err := proxy.NewHandler(cfg, logger)
	if err != nil {
		logger.Error("Failed to create handler", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/metrics", handler.Metrics)
	mux.HandleFunc("/", handler.ServeHTTP)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("Server starting", map[string]interface{}{
			"address": server.Addr,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Server exited", nil)
}

func loadConfigWithFallback(configPath string) (*config.Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file %s not found, creating default configuration\n", configPath)
		defaultCfg := config.DefaultConfig()

		if err := os.MkdirAll("config", 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		file, err := os.Create(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config file: %w", err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(defaultCfg); err != nil {
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}

		fmt.Printf("Default configuration created at %s\n", configPath)
		return defaultCfg, nil
	}

	return config.LoadConfig(configPath)
}