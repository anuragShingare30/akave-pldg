package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
	"github.com/akave-ai/akavelog/internal/model"
	"github.com/akave-ai/akavelog/internal/repository"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// InputHandler handles /inputs and /inputs/types. It uses infrastructure inputs
// and the input repository; it does not depend on Echo beyond echo.Context.
type InputHandler struct {
	Registry    *inputs.Registry
	Buffer      inputs.InputBuffer
	InputRepo   *repository.InputRepository
	Instances   map[uuid.UUID]InstanceRecord
	InstancesMu sync.Mutex
	MountIngest func(path string, h http.Handler)
}

// InstanceRecord holds a persisted input and its running MessageInput.
type InstanceRecord struct {
	Input model.Input
	Run   inputs.MessageInput
}

type inputInstanceResponse struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Title         string          `json:"title"`
	Configuration json.RawMessage `json:"configuration"`
	CreatedAt     string          `json:"created_at"`
	State         string          `json:"state"`
}

type createInputRequest struct {
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Listen      string          `json:"listen"`
	Config      json.RawMessage `json:"config"`
}

// ListTypes returns registered input type names (GET /inputs/types).
func (h *InputHandler) ListTypes(c echo.Context) error {
	types := h.Registry.ListRegistered()
	sort.Strings(types)
	return c.JSON(http.StatusOK, map[string]any{"types": types})
}

// GetAllTypesInfo returns config spec for every registered input type (GET /inputs/info).
func (h *InputHandler) GetAllTypesInfo(c echo.Context) error {
	all := h.Registry.AllTypesInfo()
	return c.JSON(http.StatusOK, map[string]any{"types": all})
}

// GetTypeInfo returns config spec for one input type (GET /inputs/types/:type).
func (h *InputHandler) GetTypeInfo(c echo.Context) error {
	typeName := c.Param("type")
	if typeName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing type in path"})
	}
	info, ok := h.Registry.GetTypeInfo(typeName)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "unknown input type: " + typeName})
	}
	return c.JSON(http.StatusOK, info)
}

// ListInputs returns all inputs from the database (GET /inputs).
func (h *InputHandler) ListInputs(c echo.Context) error {
	list, err := h.InputRepo.List(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "list inputs: " + err.Error()})
	}
	out := make([]inputInstanceResponse, 0, len(list))
	h.InstancesMu.Lock()
	for _, in := range list {
		rec, running := h.Instances[in.ID]
		state := string(in.DesiredState)
		if running && rec.Run != nil {
			state = "RUNNING"
		}
		out = append(out, inputInstanceResponse{
			ID:            in.ID.String(),
			Type:          in.Type,
			Title:         in.Title,
			Configuration: in.Configuration,
			CreatedAt:     in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			State:         state,
		})
	}
	h.InstancesMu.Unlock()
	return c.JSON(http.StatusOK, map[string]any{"inputs": out})
}

// CreateInput creates an input, persists it, and starts it (POST /inputs).
func (h *InputHandler) CreateInput(c echo.Context) error {
	var req createInputRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing 'type'"})
	}
	if req.Title == "" {
		req.Title = "input-" + uuid.New().String()[:8]
	}

	cfg := make(inputs.Config)
	if len(req.Config) > 0 {
		_ = json.Unmarshal(req.Config, &cfg)
	}
	if req.Description == "" && req.Type == "http" {
		req.Description = "raw"
	}
	if req.Description != "" {
		cfg["description"] = req.Description
	}
	if _, ok := cfg["base_path"]; !ok {
		cfg["base_path"] = "/ingest"
	}
	if req.Listen != "" {
		cfg["listen"] = req.Listen
	}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "build config: " + err.Error()})
	}

	in := model.Input{
		Type:          req.Type,
		Title:         req.Title,
		Configuration: cfgJSON,
		DesiredState:  model.InputStateRunning,
	}
	if err := h.InputRepo.Create(c.Request().Context(), &in); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "create input: " + err.Error()})
	}

	run, err := h.Registry.Create(req.Type, cfg, h.Buffer)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "create input runtime: " + err.Error()})
	}
	if err := run.Start(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "start input: " + err.Error()})
	}

	if _, hasListen := cfg["listen"]; !hasListen {
		if ep, ok := run.(inputs.HTTPEndpointInput); ok {
			path := strings.TrimPrefix(ep.Path(), "/ingest")
			if path == "" {
				path = "/"
			}
			if h.MountIngest != nil {
				h.MountIngest(path, ep.Handler())
			}
		}
	}

	h.InstancesMu.Lock()
	h.Instances[in.ID] = InstanceRecord{Input: in, Run: run}
	h.InstancesMu.Unlock()

	return c.JSON(http.StatusCreated, inputInstanceResponse{
		ID:            in.ID.String(),
		Type:          in.Type,
		Title:         in.Title,
		Configuration: in.Configuration,
		CreatedAt:     in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		State:         "RUNNING",
	})
}
