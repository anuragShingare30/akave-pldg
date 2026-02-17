package httpinput

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
)

const maxLoggedBody = 2048

// Input is an HTTP ingest endpoint that writes request body to an InputBuffer.
// It does not depend on the backend (database, model, handler, etc.).
type Input struct {
	path       string
	listenAddr string
	buffer     inputs.InputBuffer
	server     *http.Server
}

// NewInput creates an HTTP input. listenAddr is optional; if set, Start() binds to that address.
func NewInput(
	basePath string,
	description string,
	buffer inputs.InputBuffer,
	listenAddr string,
) *Input {
	basePath = "/" + strings.Trim(strings.TrimSpace(basePath), "/")
	desc := strings.TrimSpace(description)
	desc = strings.Trim(desc, "/")
	path := basePath + "/" + desc
	return &Input{
		path:       path,
		listenAddr: listenAddr,
		buffer:     buffer,
	}
}

func (i *Input) Path() string { return i.path }

func (i *Input) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		preview := string(body)
		if len(preview) > maxLoggedBody {
			preview = preview[:maxLoggedBody] + "..."
		}
		log.Printf("[ingest] received %d bytes: %s", len(body), preview)
		i.buffer.Insert(body)
		w.WriteHeader(http.StatusAccepted)
	})
}

func (i *Input) Start() error {
	if i.listenAddr == "" {
		return nil
	}
	i.server = &http.Server{
		Addr:    i.listenAddr,
		Handler: i.Handler(),
	}
	go func() {
		if err := i.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ingest] listener %s: %v", i.listenAddr, err)
		}
	}()
	log.Printf("[ingest] listening on %s", i.listenAddr)
	return nil
}

func (i *Input) Stop() error {
	if i.server != nil {
		return i.server.Close()
	}
	return nil
}
