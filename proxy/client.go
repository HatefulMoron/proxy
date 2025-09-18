package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"proxy/config"
)

type Client struct {
	httpClient *http.Client
	config     *config.Config
}

func NewClient(cfg *config.Config) (*Client, error) {
	proxyURL, err := url.Parse(cfg.Proxy.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			proxyURL.User = url.UserPassword(cfg.Proxy.Username, cfg.Proxy.Password)
			return proxyURL, nil
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Client{
		httpClient: client,
		config:     cfg,
	}, nil
}

func (c *Client) ForwardRequest(ctx context.Context, originalReq *http.Request) (*http.Response, error) {
	targetURL := fmt.Sprintf("%s://%s%s", c.config.Target.Scheme, c.config.Target.Host, originalReq.URL.Path)
	if originalReq.URL.RawQuery != "" {
		targetURL += "?" + originalReq.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(ctx, originalReq.Method, targetURL, originalReq.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	copyHeaders(req, originalReq)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxy request failed: %w", err)
	}

	return resp, nil
}

func copyHeaders(dst, src *http.Request) {
	hopHeaders := map[string]bool{
		"Connection":          true,
		"Proxy-Connection":    true,
		"Keep-Alive":         true,
		"Proxy-Authenticate": true,
		"Proxy-Authorization": true,
		"Te":                 true,
		"Trailer":            true,
		"Transfer-Encoding":  true,
		"Upgrade":            true,
	}

	for name, values := range src.Header {
		if !hopHeaders[name] {
			for _, value := range values {
				dst.Header.Add(name, value)
			}
		}
	}

	dst.Header.Set("X-Forwarded-For", src.RemoteAddr)
	dst.Header.Set("X-Forwarded-Proto", "http")
	if src.Host != "" {
		dst.Header.Set("X-Forwarded-Host", src.Host)
	}
}

