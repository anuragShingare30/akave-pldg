package httpinput

import (
	"fmt"

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
		Description: "HTTP ingest endpoint. Accepts POST body and writes to the log buffer. Can be mounted on the main server or bound to an external port.",
		Fields: []inputs.ConfigField{
			{Name: "description", Type: "string", Required: true, Description: "Path segment for the endpoint (e.g. 'raw' → /ingest/raw)", Example: "raw"},
			{Name: "base_path", Type: "string", Required: false, Description: "Base path prefix", Example: "/ingest"},
			{Name: "listen", Type: "string", Required: false, Description: "Optional host:port to bind (e.g. :9001 or 0.0.0.0:9001). If set, input listens on this address instead of being mounted on the main server.", Example: ":9001"},
		},
	}
}

func (f *Factory) Create(cfg inputs.Config, buffer inputs.InputBuffer) (inputs.MessageInput, error) {
	description, _ := cfg["description"].(string)
	if description == "" {
		return nil, fmt.Errorf("missing 'description' for http input")
	}
	basePath, _ := cfg["base_path"].(string)
	if basePath == "" {
		basePath = "/inputs"
	}
	listen, _ := cfg["listen"].(string)
	return NewInput(basePath, description, buffer, listen), nil
}
