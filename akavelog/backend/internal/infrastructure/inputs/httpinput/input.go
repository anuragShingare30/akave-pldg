package httpinput

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
	"github.com/akave-ai/akavelog/internal/model"
)

const maxLoggedBody = 2048
const maxBodyInRawLog = 64 * 1024 // 64KB max body stored in raw_request

// Input is an HTTP ingest endpoint that writes request body to an InputBuffer.
// It also logs the full HTTP request (method, path, query, headers, body) as a raw log entry.
type Input struct {
	path       string
	listenAddr string
	buffer     inputs.InputBuffer
	server     *http.Server
}

// NewInput creates an HTTP input. listenAddr is optional; if set, Start() binds to that address
// and the path is just basePath (e.g. /ingest). Otherwise path is basePath/description (e.g. /ingest/raw).
func NewInput(
	basePath string,
	description string,
	buffer inputs.InputBuffer,
	listenAddr string,
) *Input {
	basePath = "/" + strings.Trim(strings.TrimSpace(basePath), "/")
	if basePath == "/" {
		basePath = "/ingest"
	}
	var path string
	if listenAddr != "" {
		path = basePath
	} else {
		desc := strings.TrimSpace(description)
		desc = strings.Trim(desc, "/")
		if desc == "" {
			desc = "raw"
		}
		path = basePath + "/" + desc
	}
	return &Input{
		path:       path,
		listenAddr: listenAddr,
		buffer:     buffer,
	}
}

func (i *Input) Path() string { return i.path }

func corsHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (i *Input) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		// Build full request data for raw log (method, path, query, headers, body)
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		bodyStr := string(body)
		if len(bodyStr) > maxBodyInRawLog {
			bodyStr = bodyStr[:maxBodyInRawLog] + "... [truncated]"
		}
		rawReq := &model.RawRequestData{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Headers: headers,
			Body:    bodyStr,
		}
		entry := model.LogEntry{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Service:    "ingest",
			Level:      "info",
			Message:    "raw http request",
			Tags:       map[string]string{"path": r.URL.Path},
			RawRequest: rawReq,
		}
		rawLogJSON, err := json.Marshal(entry)
		if err != nil {
			log.Printf("[ingest] marshal raw log: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		i.buffer.Insert(rawLogJSON)

		// If body present, also insert as-is so normal log payloads are still ingested
		if len(body) > 0 {
			preview := string(body)
			if len(preview) > maxLoggedBody {
				preview = preview[:maxLoggedBody] + "..."
			}
			log.Printf("[ingest] received %d bytes: %s", len(body), preview)
			i.buffer.Insert(body)
		}

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
