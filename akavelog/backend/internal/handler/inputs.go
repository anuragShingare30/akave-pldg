package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
	"github.com/akave-ai/akavelog/internal/model"
	"github.com/akave-ai/akavelog/internal/repository"
	"github.com/akave-ai/akavelog/internal/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// InputHandler handles /inputs and /inputs/types. It uses infrastructure inputs
// and the input repository; it does not depend on Echo beyond echo.Context.
type InputHandler struct {
	Registry      *inputs.Registry
	Buffer        inputs.InputBuffer
	InputRepo     *repository.InputRepository
	Instances     map[uuid.UUID]InstanceRecord
	InstancesMu   sync.Mutex
	MountIngest   func(path string, h http.Handler)
	UnmountIngest func(path string)
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
	return response.OK(c, map[string]any{"types": types}, "")
}

// GetAllTypesInfo returns config spec for every registered input type (GET /inputs/info).
func (h *InputHandler) GetAllTypesInfo(c echo.Context) error {
	all := h.Registry.AllTypesInfo()
	return response.OK(c, map[string]any{"types": all}, "")
}

// GetTypeInfo returns config spec for one input type (GET /inputs/types/:type).
func (h *InputHandler) GetTypeInfo(c echo.Context) error {
	typeName := c.Param("type")
	if typeName == "" {
		return response.BadRequest(c, "missing type in path", "missing type in path")
	}
	info, ok := h.Registry.GetTypeInfo(typeName)
	if !ok {
		return response.NotFound(c, "unknown input type", "unknown input type: "+typeName)
	}
	return response.OK(c, info, "")
}

// ListInputs returns all inputs from the database (GET /inputs).
func (h *InputHandler) ListInputs(c echo.Context) error {
	list, err := h.InputRepo.List(c.Request().Context())
	if err != nil {
		return response.InternalError(c, "list inputs failed", "list inputs: "+err.Error())
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
	return response.OK(c, map[string]any{"inputs": out}, "")
}

// CreateInput creates an input, persists it, and starts it (POST /inputs).
func (h *InputHandler) CreateInput(c echo.Context) error {
	var req createInputRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body", "invalid JSON body")
	}
	if req.Type == "" {
		return response.BadRequest(c, "missing type", "missing 'type'")
	}
	if req.Title == "" {
		req.Title = "input-" + uuid.New().String()[:8]
	}

	cfg := make(inputs.Config)
	if len(req.Config) > 0 {
		_ = json.Unmarshal(req.Config, &cfg)
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
	if req.Type == "http" && cfg["listen"] == nil {
		return response.BadRequest(c, "listen is required", "http input must have a listen port (e.g. :9001); nothing is mounted on the main server")
	}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return response.BadRequest(c, "invalid config", "build config: "+err.Error())
	}

	// Validate config via factory
	if err := h.Registry.ValidateConfig(req.Type, cfg); err != nil {
		return response.BadRequest(c, "invalid config", err.Error())
	}

	// For http: ensure the same port is not already in use
	if req.Type == "http" {
		listen, _ := cfg["listen"].(string)
		existing, err := h.InputRepo.List(c.Request().Context())
		if err != nil {
			return response.InternalError(c, "list inputs failed", "list inputs: "+err.Error())
		}
		for _, ex := range existing {
			if ex.Type != "http" {
				continue
			}
			var exCfg map[string]interface{}
			if len(ex.Configuration) > 0 {
				_ = json.Unmarshal(ex.Configuration, &exCfg)
			}
			if exListen, _ := exCfg["listen"].(string); exListen != "" && exListen == listen {
				return response.Error(c, 409, "listen address already in use", "listen "+listen+" is already used by another input")
			}
		}
	}

	in := model.Input{
		Type:          req.Type,
		Title:         req.Title,
		Configuration: cfgJSON,
		DesiredState:  model.InputStateRunning,
	}
	if err := h.InputRepo.Create(c.Request().Context(), &in); err != nil {
		return response.InternalError(c, "create input failed", "create input: "+err.Error())
	}

	run, err := h.Registry.Create(req.Type, cfg, h.Buffer)
	if err != nil {
		return response.BadRequest(c, "create input runtime failed", "create input runtime: "+err.Error())
	}
	if err := run.Start(); err != nil {
		return response.InternalError(c, "start input failed", "start input: "+err.Error())
	}

	// No mounting on main server: each input runs on its own listen port only

	h.InstancesMu.Lock()
	h.Instances[in.ID] = InstanceRecord{Input: in, Run: run}
	h.InstancesMu.Unlock()

	return response.Created(c, inputInstanceResponse{
		ID:            in.ID.String(),
		Type:          in.Type,
		Title:         in.Title,
		Configuration: in.Configuration,
		CreatedAt:     in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		State:         "RUNNING",
	}, "input created")
}

// stopAndUnmount stops the running input and unmounts its path if it is an HTTP endpoint.
func (h *InputHandler) stopAndUnmount(rec InstanceRecord) {
	if rec.Run != nil {
		_ = rec.Run.Stop()
	}
	if rec.Run != nil {
		if ep, ok := rec.Run.(inputs.HTTPEndpointInput); ok && h.UnmountIngest != nil {
			path := strings.TrimPrefix(ep.Path(), "/ingest")
			if path == "" {
				path = "/"
			}
			h.UnmountIngest(path)
		}
	}
}

// UpdateInput updates an input by id (PUT /inputs/:id). Stops existing instance, updates DB, restarts.
func (h *InputHandler) UpdateInput(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid id", "invalid id")
	}

	var req createInputRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "invalid request body", "invalid JSON body")
	}

	in, err := h.InputRepo.GetByID(c.Request().Context(), id)
	if err != nil || in == nil {
		return response.NotFound(c, "input not found", "input not found")
	}

	// Stop and unmount existing instance if running
	h.InstancesMu.Lock()
	rec, running := h.Instances[in.ID]
	if running {
		h.stopAndUnmount(rec)
		delete(h.Instances, in.ID)
	}
	h.InstancesMu.Unlock()

	// Build new config (same as CreateInput)
	if req.Title != "" {
		in.Title = req.Title
	}
	cfg := make(inputs.Config)
	if len(in.Configuration) > 0 {
		_ = json.Unmarshal(in.Configuration, &cfg)
	}
	if len(req.Config) > 0 {
		_ = json.Unmarshal(req.Config, &cfg)
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
		return response.BadRequest(c, "invalid config", "build config: "+err.Error())
	}

	// Validate config via factory
	if err := h.Registry.ValidateConfig(in.Type, cfg); err != nil {
		return response.BadRequest(c, "invalid config", err.Error())
	}
	// For http with listen: ensure port not already in use by another input (excluding this one)
	if in.Type == "http" {
		if listen, _ := cfg["listen"].(string); listen != "" {
			existing, err := h.InputRepo.List(c.Request().Context())
			if err != nil {
				return response.InternalError(c, "list inputs failed", "list inputs: "+err.Error())
			}
			for _, ex := range existing {
				if ex.ID == in.ID || ex.Type != "http" {
					continue
				}
				var exCfg map[string]interface{}
				if len(ex.Configuration) > 0 {
					_ = json.Unmarshal(ex.Configuration, &exCfg)
				}
				if exListen, _ := exCfg["listen"].(string); exListen != "" && exListen == listen {
					return response.Error(c, 409, "listen address already in use", "listen "+listen+" is already used by another input")
				}
			}
		}
	}

	in.Configuration = cfgJSON
	in.DesiredState = model.InputStateRunning
	if err := h.InputRepo.Update(c.Request().Context(), in); err != nil {
		return response.InternalError(c, "update input failed", "update input: "+err.Error())
	}

	run, err := h.Registry.Create(in.Type, cfg, h.Buffer)
	if err != nil {
		return response.BadRequest(c, "create input runtime failed", "create input runtime: "+err.Error())
	}
	if err := run.Start(); err != nil {
		return response.InternalError(c, "start input failed", "start input: "+err.Error())
	}
	h.InstancesMu.Lock()
	h.Instances[in.ID] = InstanceRecord{Input: *in, Run: run}
	h.InstancesMu.Unlock()

	return response.OK(c, inputInstanceResponse{
		ID:            in.ID.String(),
		Type:          in.Type,
		Title:         in.Title,
		Configuration: in.Configuration,
		CreatedAt:     in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		State:         "RUNNING",
	}, "input updated")
}

// DeleteInput deletes an input by id (DELETE /inputs/:id). Stops and unmounts then removes from DB.
func (h *InputHandler) DeleteInput(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid id", "invalid id")
	}

	in, err := h.InputRepo.GetByID(c.Request().Context(), id)
	if err != nil || in == nil {
		return response.NotFound(c, "input not found", "input not found")
	}

	h.InstancesMu.Lock()
	rec, running := h.Instances[id]
	if running {
		h.stopAndUnmount(rec)
		delete(h.Instances, id)
	}
	h.InstancesMu.Unlock()

	if err := h.InputRepo.Delete(c.Request().Context(), id); err != nil {
		return response.InternalError(c, "delete input failed", "delete input: "+err.Error())
	}
	return response.OK(c, nil, "input deleted")
}

// RestoreInputs loads inputs from the DB and starts each on its listen port. Nothing is mounted on the main server.
func (h *InputHandler) RestoreInputs(ctx context.Context) {
	list, err := h.InputRepo.List(ctx)
	if err != nil {
		log.Printf("[inputs] restore list: %v", err)
		return
	}
	for _, in := range list {
		if in.Type != "http" {
			continue
		}
		cfg := make(inputs.Config)
		if len(in.Configuration) > 0 {
			_ = json.Unmarshal(in.Configuration, &cfg)
		}
		if _, hasListen := cfg["listen"]; !hasListen {
			log.Printf("[inputs] skip restore %s: no listen (inputs must have own port)", in.Title)
			continue
		}
		if _, ok := cfg["base_path"]; !ok {
			cfg["base_path"] = "/ingest"
		}
		run, err := h.Registry.Create(in.Type, cfg, h.Buffer)
		if err != nil {
			log.Printf("[inputs] restore create %s: %v", in.Title, err)
			continue
		}
		if err := run.Start(); err != nil {
			log.Printf("[inputs] restore start %s: %v", in.Title, err)
			continue
		}
		h.InstancesMu.Lock()
		h.Instances[in.ID] = InstanceRecord{Input: in, Run: run}
		h.InstancesMu.Unlock()
		log.Printf("[inputs] restored %s → listen %s", in.Title, cfg["listen"])
	}
}
