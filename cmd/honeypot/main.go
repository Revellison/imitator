package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go_videostream/config"
	"go_videostream/core"
	"go_videostream/memory"
	"go_videostream/modules/cache_warmer"
	"go_videostream/modules/videostream"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	log.Println("Starting Go-Honeypot...")

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ramCache := memory.NewCache()

	registry := core.NewRegistry()
	registry.Register(videostream.New())
	registry.Register(cache_warmer.New())

	getConfig := func(name string) interface{} {
		switch name {
		case "videostream":
			return &cfg.Modules.VideoStream
		case "cache_warmer":
			return &cfg.Modules.CacheWarmer
		}
		return nil
	}

	if err := registry.InitModules(cfg.EnabledModules, getConfig, ramCache); err != nil {
		log.Fatalf("Failed to initialize modules: %v", err)
	}

	mux := http.NewServeMux()
	registry.RegisterAllRoutes(mux)

	srv := &http.Server{
		Addr:           cfg.Server.Listen,
		Handler:        mux,
		ReadTimeout:    cfg.Server.ReadTimeout.Duration,
		WriteTimeout:   cfg.Server.WriteTimeout.Duration,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	go func() {
		log.Printf("Server listening on %s", cfg.Server.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	registry.ShutdownAll()

	log.Println("Honeypot exited gracefully")
}
