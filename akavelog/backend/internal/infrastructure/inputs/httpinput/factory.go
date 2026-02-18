package httpinput

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
)

// Factory creates HTTP ingest inputs. Registers as "http".
type Factory struct{}

func (f *Factory) Name() string {
	return "http"
}

func (f *Factory) ConfigSpec() inputs.InputTypeInfo {
	return inputs.InputTypeInfo{
		Type:        "http",
		Description: "HTTP ingest endpoint on its own port. Each input listens on host:port and serves POST /ingest. Nothing is mounted on the main server.",
		Fields: []inputs.ConfigField{
			{Name: "listen", Type: "string", Required: true, Description: "host:port to bind (e.g. :9001). Must be unique across inputs.", Example: ":9001"},
			{Name: "base_path", Type: "string", Required: false, Description: "Path served on the listen port", Example: "/ingest"},
		},
	}
}

// ValidateConfig validates http input config. Listen is required (each input has its own port).
func (f *Factory) ValidateConfig(cfg inputs.Config) error {
	listen, _ := cfg["listen"].(string)
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return fmt.Errorf("listen is required: each input must have its own port (e.g. :9001)")
	}
	if !validListenAddr(listen) {
		return fmt.Errorf("listen must be host:port or :port (e.g. :9001 or 0.0.0.0:9001)")
	}
	return nil
}

func validListenAddr(addr string) bool {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return false
	}
	port := addr[idx+1:]
	if len(port) == 0 || len(port) > 5 {
		return false
	}
	for _, r := range port {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func (f *Factory) Create(cfg inputs.Config, buffer inputs.InputBuffer) (inputs.MessageInput, error) {
	listen, _ := cfg["listen"].(string)
	if strings.TrimSpace(listen) == "" {
		return nil, fmt.Errorf("listen is required for http input")
	}
	basePath, _ := cfg["base_path"].(string)
	if basePath == "" {
		basePath = "/ingest"
	}
	return NewInput(basePath, "", buffer, listen), nil
}
