package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"proxy/config"
	"proxy/logging"
)

type Handler struct {
	client *Client
	logger *logging.Logger
	config *config.Config
}

func NewHandler(cfg *config.Config, logger *logging.Logger) (*Handler, error) {
	if cfg.Proxy.URL == "" {
		return nil, fmt.Errorf("proxy URL is required")
	}

	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Handler{
		client: client,
		logger: logger,
		config: cfg,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	h.logger.Info("Request received", map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"query":      r.URL.RawQuery,
		"user_agent": r.Header.Get("User-Agent"),
		"remote_ip":  r.RemoteAddr,
	})

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := h.client.ForwardRequest(ctx, r)
	if err != nil {
		statusCode := http.StatusBadGateway
		errorMsg := "Proxy error"

		if ctx.Err() == context.DeadlineExceeded {
			statusCode = http.StatusGatewayTimeout
			errorMsg = "Request timeout"
		}

		h.logger.Error("Failed to forward request", map[string]interface{}{
			"error":       err.Error(),
			"path":        r.URL.Path,
			"method":      r.Method,
			"elapsed":     time.Since(start),
			"status_code": statusCode,
			"timeout":     ctx.Err() == context.DeadlineExceeded,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(fmt.Sprintf(`{"error":"%s","timestamp":"%s"}`, errorMsg, time.Now().Format(time.RFC3339))))
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)

	written, err := io.Copy(w, resp.Body)
	if err != nil {
		h.logger.Error("Failed to copy response body", map[string]interface{}{
			"error":   err.Error(),
			"path":    r.URL.Path,
			"method":  r.Method,
			"elapsed": time.Since(start),
		})
		return
	}

	h.logger.Info("Request completed", map[string]interface{}{
		"method":       r.Method,
		"path":         r.URL.Path,
		"status_code":  resp.StatusCode,
		"bytes_written": written,
		"elapsed":      time.Since(start),
		"content_type": resp.Header.Get("Content-Type"),
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	testReq, err := http.NewRequestWithContext(ctx, "HEAD", h.config.Target.Scheme+"://"+h.config.Target.Host, nil)
	if err != nil {
		h.logger.Error("Health check failed to create request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Health check failed", http.StatusServiceUnavailable)
		return
	}

	_, err = h.client.httpClient.Do(testReq)
	if err != nil {
		h.logger.Error("Health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Health check failed", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"metrics":{"uptime":"` + time.Since(time.Now()).String() + `","requests_total":0}}`))
}

func copyResponseHeaders(dst http.ResponseWriter, src *http.Response) {
	hopHeaders := map[string]bool{
		"Connection":        true,
		"Proxy-Connection":  true,
		"Keep-Alive":       true,
		"Proxy-Authenticate": true,
		"Proxy-Authorization": true,
		"Te":               true,
		"Trailer":          true,
		"Transfer-Encoding": true,
		"Upgrade":          true,
	}

	for name, values := range src.Header {
		if !hopHeaders[name] {
			for _, value := range values {
				dst.Header().Add(name, value)
			}
		}
	}
}