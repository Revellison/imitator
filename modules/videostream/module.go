package videostream

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go_videostream/config"
	"go_videostream/memory"
)

type VideoStreamModule struct {
	cfg   *config.VideoStreamConfig
	cache *memory.Cache
}

func New() *VideoStreamModule {
	return &VideoStreamModule{}
}

func (m *VideoStreamModule) Name() string {
	return "videostream"
}

func (m *VideoStreamModule) Init(cfg interface{}, ramCache interface{}) error {
	m.cfg = cfg.(*config.VideoStreamConfig)
	m.cache = ramCache.(*memory.Cache)

	if m.cfg.PreloadToRAM {
		if err := m.cache.LoadFromDirectory(m.cfg.ChunksDir); err != nil {
			return err
		}
	}

	return nil
}

func (m *VideoStreamModule) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(m.cfg.Path, m.handleRequest)
}

func (m *VideoStreamModule) Shutdown() error {
	return nil
}

func (m *VideoStreamModule) handleRequest(w http.ResponseWriter, r *http.Request) {
	for k, v := range m.cfg.Headers {
		w.Header().Set(k, v)
	}

	var data []byte

	if m.cfg.DeliveryMode == "sequential" {
		id := r.URL.Query().Get(m.cfg.SeqKey)
		if id != "" {
			if m.cfg.PreloadToRAM {
				d, ok := m.cache.Get(id)
				if ok {
					data = d
				}
			} else {
				path := filepath.Join(m.cfg.ChunksDir, id+".ts")
				d, err := os.ReadFile(path)
				if err == nil {
					data = d
				}
			}
		}
	}

	if data == nil && (m.cfg.FallbackMode == "random" || m.cfg.DeliveryMode == "random") {
		if m.cfg.PreloadToRAM {
			data = m.cache.GetRandom(func(max int) int {
				n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
				return int(n.Int64())
			})
		} else {
			// Very basic random from disk for fallback (reading dir each time)
			entries, _ := os.ReadDir(m.cfg.ChunksDir)
			var tsFiles []string
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".ts") {
					tsFiles = append(tsFiles, e.Name())
				}
			}
			if len(tsFiles) > 0 {
				n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(tsFiles))))
				file := tsFiles[n.Int64()]
				d, _ := os.ReadFile(filepath.Join(m.cfg.ChunksDir, file))
				data = d
			}
		}
	}

	if data != nil {
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
