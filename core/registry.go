package core

import (
	"fmt"
	"log"
	"net/http"

	"go_videostream/modules"
)

// Registry manages the lifecycle of module plugins
type Registry struct {
	modules map[string]modules.Module
	active  []modules.Module
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]modules.Module),
		active:  make([]modules.Module, 0),
	}
}

// Register adds a module to the registry memory
func (r *Registry) Register(m modules.Module) {
	r.modules[m.Name()] = m
}

// InitModules initializes specifically enabled modules
func (r *Registry) InitModules(enabled []string, getCfg func(string) interface{}, ramCache interface{}) error {
	for _, name := range enabled {
		m, exists := r.modules[name]
		if !exists {
			return fmt.Errorf("module %s not found in registry", name)
		}

		cfg := getCfg(name)
		err := m.Init(cfg, ramCache)
		if err != nil {
			return fmt.Errorf("failed to init module %s: %w", name, err)
		}

		r.active = append(r.active, m)
		log.Printf("Module successfully initialized: %s", name)
	}
	return nil
}

// RegisterAllRoutes attaches active module handlers to main mux
func (r *Registry) RegisterAllRoutes(mux *http.ServeMux) {
	for _, m := range r.active {
		m.RegisterRoutes(mux)
	}
}

// ShutdownAll gracefully shuts down all active modules
func (r *Registry) ShutdownAll() {
	for i := len(r.active) - 1; i >= 0; i-- {
		m := r.active[i]
		log.Printf("Shutting down module: %s", m.Name())
		if err := m.Shutdown(); err != nil {
			log.Printf("Error shutting down module %s: %v", m.Name(), err)
		}
	}
}
