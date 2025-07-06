package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/tekig/photo-backup-server/internal/gateway/http"
	"github.com/tekig/photo-backup-server/internal/photo"
	"github.com/tekig/photo-backup-server/internal/repository/cmd"
	"github.com/tekig/photo-backup-server/internal/repository/s3"
	"gopkg.in/yaml.v2"
)

type App struct {
	gateway *http.Gateway
}

func New() (*App, error) {
	configFile := flag.String("config", "./config.yaml", "config")
	flag.Parse()

	configData, err := os.ReadFile(*configFile)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	thumbnails := cmd.New()
	storage, err := s3.New(s3.StorageConfig{
		Endpoint:     config.Storage.Endpoint,
		AccessKey:    config.Storage.AccessKey,
		AccessSecret: config.Storage.AccessSecret,
		Region:       config.Storage.Region,
		Bucket:       config.Storage.Bucket,
	})
	if err != nil {
		return nil, fmt.Errorf("new s3 storage: %w", err)
	}

	usecase, err := photo.New(storage, thumbnails)
	if err != nil {
		return nil, fmt.Errorf("new photo: %w", err)
	}

	gateway := http.New(http.GatewayConfig{
		Photo:   usecase,
		Address: config.Gateway.Address,
	})

	return &App{
		gateway: gateway,
	}, nil
}

func (a *App) Run() error {
	if err := a.gateway.Run(); err != nil {
		return fmt.Errorf("gateway run: %w", err)
	}

	return nil
}

func (a *App) Shutdown() error {
	if err := a.gateway.Shutdown(); err != nil {
		return fmt.Errorf("yas3trigger shutdown: %w", err)
	}

	return nil
}
