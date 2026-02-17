package httpinput

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
)

type memBuffer struct {
	mu   sync.Mutex
	msgs [][]byte
}

func (b *memBuffer) Insert(p []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]byte, len(p))
	copy(cp, p)
	b.msgs = append(b.msgs, cp)
}

func (b *memBuffer) Last() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.msgs) == 0 {
		return nil
	}
	return b.msgs[len(b.msgs)-1]
}

func TestHTTPInput_InsertsBodyIntoBuffer(t *testing.T) {
	reg := inputs.NewRegistry()
	reg.Register(&Factory{})

	buf := &memBuffer{}
	mux := http.NewServeMux()
	specs := []inputs.InputSpec{
		{Type: "http", Description: "raw-http", Config: inputs.Config{"base_path": "/ingest"}},
	}
	if err := reg.MountHTTPEndpoints(mux, specs, buf); err != nil {
		t.Fatalf("mount endpoints: %v", err)
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := []byte("hello raw http")
	resp, err := http.Post(srv.URL+"/ingest/raw-http", "text/plain", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, resp.StatusCode)
	}
	got := buf.Last()
	if !bytes.Equal(got, body) {
		t.Fatalf("expected inserted %q, got %q", string(body), string(got))
	}
}
