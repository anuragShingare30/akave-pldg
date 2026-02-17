package server

import (
	"context"
	"log"
	"sort"
	"sync"

	"github.com/akave-ai/akavelog/internal/config"
	"github.com/akave-ai/akavelog/internal/handler"
	"github.com/akave-ai/akavelog/internal/infrastructure/inputs"
	_ "github.com/akave-ai/akavelog/internal/infrastructure/inputs/httpinput"
	"github.com/akave-ai/akavelog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// memoryBuffer implements inputs.InputBuffer for received log payloads.
type memoryBuffer struct {
	mu   sync.Mutex
	logs [][]byte
}

func (b *memoryBuffer) Insert(p []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.logs = append(b.logs, p)
}

// Server holds the Echo app and dependencies.
type Server struct {
	Echo   *echo.Echo
	Config *config.Config
}

// New builds the Echo server and registers routes.
// Caller must provide a non-nil pool (e.g. from database.Database.Pool).
func New(cfg *config.Config, pool *pgxpool.Pool) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover(), middleware.Logger())

	buffer := &memoryBuffer{}
	ingestD := NewIngestDispatcher()

	inputHandler := &handler.InputHandler{
		Registry:    inputs.GlobalRegistry,
		Buffer:      buffer,
		InputRepo:   repository.NewInputRepository(pool),
		Instances:   make(map[uuid.UUID]handler.InstanceRecord),
		MountIngest: ingestD.Mount,
	}

	// Management API
	e.GET("/inputs/types", inputHandler.ListTypes)
	e.GET("/inputs/types/:type", inputHandler.GetTypeInfo)
	e.GET("/inputs/info", inputHandler.GetAllTypesInfo)
	e.GET("/inputs", inputHandler.ListInputs)
	e.POST("/inputs", inputHandler.CreateInput)

	// Ingest: any path under /ingest is dispatched by path
	e.Any("/ingest/*", echo.WrapHandler(ingestD))

	types := inputs.GlobalRegistry.ListRegistered()
	sort.Strings(types)
	log.Printf("Registered input types: %v", types)

	return &Server{Echo: e, Config: cfg}
}

// Start starts the HTTP server. Blocks until the context is cancelled or the server fails.
func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.Echo.Shutdown(context.Background())
	}()
	addr := ":" + s.Config.Server.Port
	return s.Echo.Start(addr)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Echo.Shutdown(ctx)
}
