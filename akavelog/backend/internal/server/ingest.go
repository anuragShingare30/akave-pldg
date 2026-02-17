package server

import (
	"net/http"
	"strings"
	"sync"
)

// IngestDispatcher routes /ingest/<path> to registered handlers.
// Handlers are registered by path segment (e.g. "raw" or "/raw").
type IngestDispatcher struct {
	mu       sync.RWMutex
	handlers map[string]http.Handler
}

// NewIngestDispatcher returns a new IngestDispatcher.
func NewIngestDispatcher() *IngestDispatcher {
	return &IngestDispatcher{
		handlers: make(map[string]http.Handler),
	}
}

// Mount registers a handler for the given path (e.g. "/raw" or "raw").
// Path is normalized to a leading slash.
func (d *IngestDispatcher) Mount(path string, h http.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	d.handlers[path] = h
}

// ServeHTTP strips the /ingest prefix and dispatches to the registered handler.
func (d *IngestDispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/ingest")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	d.mu.RLock()
	h, ok := d.handlers[path]
	d.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.ServeHTTP(w, r)
}
