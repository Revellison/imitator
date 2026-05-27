package cache_warmer

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go_videostream/config"
)

type CacheWarmerModule struct {
	cfg    *config.CacheWarmerConfig
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New() *CacheWarmerModule {
	return &CacheWarmerModule{}
}

func (m *CacheWarmerModule) Name() string {
	return "cache_warmer"
}

func (m *CacheWarmerModule) Init(cfg interface{}, ramCache interface{}) error {
	m.cfg = cfg.(*config.CacheWarmerConfig)

	if m.cfg.EnabledWorker {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		m.wg.Add(1)
		go m.runParams(ctx)
	}
	return nil
}

func (m *CacheWarmerModule) RegisterRoutes(mux *http.ServeMux) {
	// Cache warmer exposes no routes
}

func (m *CacheWarmerModule) Shutdown() error {
	if m.cancel != nil {
		m.cancel()
		m.wg.Wait()
	}
	return nil
}

func (m *CacheWarmerModule) runParams(ctx context.Context) {
	defer m.wg.Done()
	log.Println("Cache warmer started")

	ticker := time.NewTicker(time.Duration(m.cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	client := m.createClient()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cache warmer stopped")
			return
		case <-ticker.C:
			m.warmUp(client)
		}
	}
}

func (m *CacheWarmerModule) createClient() *http.Client {
	transport := &http.Transport{}

	if m.cfg.LocalProxy != "" {
		proxyURL, err := url.Parse(m.cfg.LocalProxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			log.Printf("Invalid proxy URL: %v", err)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

func (m *CacheWarmerModule) warmUp(client *http.Client) {
	sem := make(chan struct{}, m.cfg.ConcurrencyLimit)
	var wg sync.WaitGroup

	for _, target := range m.cfg.Targets {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, err := client.Get(url)
			if err != nil {
				log.Printf("Warmer error %s: %v", url, err)
				return
			}
			defer resp.Body.Close()
			log.Printf("Warmed %s (status: %d)", url, resp.StatusCode)
		}(target)
	}
	wg.Wait()
}
