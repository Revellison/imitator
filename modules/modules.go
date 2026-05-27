package modules

import "net/http"

type Module interface {
	Name() string
	Init(cfg interface{}, ramCache interface{}) error
	RegisterRoutes(mux *http.ServeMux)
	Shutdown() error
}
