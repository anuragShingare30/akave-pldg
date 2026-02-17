package config

import (
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability" validate:"required"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type DatabaseConfig struct {
	Host            string `koanf:"host" validate:"required"`
	Port            int    `koanf:"port" validate:"required"`
	User            string `koanf:"user" validate:"required"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name" validate:"required"`
	SSLMode         string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}

// LoadConfig loads the configuration from environment variables using koanf.
func LoadConfig() (mainConfig *Config, err error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")
	err = k.Load(env.Provider("AKAVELOG_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "AKAVELOG_"))
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load initial env variables")
	}

	mainConfig = &Config{}
	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not unmarshal mainconfig")
	}

	validate := validator.New()
	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not validate the struct")
	}

	// set default observability config if not provided
	// in config struct we set Observability as pointer type to check whether it is nil or not
	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultObservabilityConfig()
	}

	// fill some of the fields
	mainConfig.Observability.ServiceName = "akavelog"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	// automatic pointer dereferencing for method calls
	err = mainConfig.Observability.Validate()
	if err != nil {
		logger.Fatal().Err(err).Msg("invalid observability config")
	}

	return
}

// App holds simple app config for the Echo server (DatabaseURL, ServerAddr).
// Used by server.Server when running the inputs/ingest API.
type App struct {
	DatabaseURL string
	ServerAddr  string
}

// DefaultApp returns defaults for local development.
func DefaultApp() App {
	return App{
		DatabaseURL: "postgres://chayan:chayan@localhost:5432/akavelog?sslmode=disable",
		ServerAddr:  ":8080",
	}
}

// LoadApp reads App config from the environment (DATABASE_URL, SERVER_ADDR).
func LoadApp() App {
	app := DefaultApp()
	if v := os.Getenv("DATABASE_URL"); v != "" {
		app.DatabaseURL = v
	}
	if v := os.Getenv("SERVER_ADDR"); v != "" {
		app.ServerAddr = v
	}
	return app
}
