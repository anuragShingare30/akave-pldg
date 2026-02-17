package inputs

import "net/http"

// MessageInput is the minimal interface implemented by all input types.
// It can be started and stopped.
type MessageInput interface {
	Start() error
	Stop() error
}

// HTTPEndpointInput is implemented by inputs that expose an HTTP endpoint.
// They provide a path and handler that can be mounted on any HTTP router.
type HTTPEndpointInput interface {
	MessageInput
	Path() string
	Handler() http.Handler
}
