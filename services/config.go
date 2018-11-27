package services

import (
	"context"
	"encoding/json"
	"os"

	"cloud.google.com/go/logging"
	"github.com/pkg/errors"
)

const (
	logName = "butler"
)

// Config defines parameters that will be used to start the
// service and what targets to listen for.
type Config struct {
	ListenAddress string            `json:"listenAddress,omitempty"`
	TLS           *TLS              `json:"tls,omitempty"`
	Targets       map[string]string `json:"targets,omitempty"`
	Logger        *logging.Logger
	ProjectID     string
}

// ReadConfig pulls the configuration from either a file parameter or
// a reference to an environment variable.
func ReadConfig(file *string, envVar *string) (*Config, error) {
	var cfg *Config
	var err error
	switch {
	case file != nil && *file != "":
		cfg, err = fromFile(*file)
	case envVar != nil && *envVar != "":
		cfg, err = fromEnv(*envVar)
	default:
		return nil, errors.New("file or environment variable is required")
	}

	if err != nil {
		return nil, err
	}

	cfg.ProjectID = os.Getenv("PROJECT_ID")

	ctx := context.Background()

	// Creates a client.
	client, err := logging.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	cfg.Logger = client.Logger(logName)

	return cfg, nil
}

func fromFile(file string) (*Config, error) {
	if file == "" {
		return nil, errors.New("invalid configuration file")
	}

	if _, err := os.Stat(file); err != nil {
		return nil, errors.Errorf("failed to read file: %s", file)
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Errorf("failed to read config file: %v", err)
	}

	var cfg *Config
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, errors.Errorf("failed to decode JSON: %v", err)
	}

	return cfg, nil
}

func fromEnv(envVar string) (*Config, error) {
	if envVar == "" {
		return nil, errors.New("invalid configuration variable")
	}

	data := os.Getenv(envVar)

	if data == "" {
		return nil, errors.Errorf("failed to read environment variable: %s", envVar)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, errors.Errorf("failed to decode JSON: %v", err)
	}

	return &cfg, nil
}
