package app

import (
	"fmt"
	"log/slog"
	"os"

	yas3trigger "github.com/tekig/photo-backup-server/internal/gateway/ya-s3-trigger"
	"github.com/tekig/photo-backup-server/internal/photo"
)

type App struct {
	yas3trigger *yas3trigger.HTTP
}

func env(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("empty env `%s`", key))
	}

	return v
}

func New() (*App, error) {
	app := &App{}

	p, err := photo.New(photo.Config{
		Endpoint:     env("ENDPOINT"),
		AccessKey:    env("ACCESS_KEY"),
		AccessSecret: env("ACCESS_SECRET"),
		Region:       env("REGION"),
	})
	if err != nil {
		return nil, fmt.Errorf("photo: %w", err)
	}

	switch value := env("GATEWAY"); value {
	case "YA-S3-TRIGGER":
		g, err := yas3trigger.NewHTTP(yas3trigger.ConfigHTTP{
			Listen: ":" + env("PORT"),
			Photo:  p,
			Logger: slog.Default(),
		})
		if err != nil {
			return nil, fmt.Errorf("yas3trigger: %w", err)
		}

		app.yas3trigger = g
	default:
		return nil, fmt.Errorf("unknown gateway `%s`", value)
	}

	return app, nil
}

func (a *App) Run() error {
	if err := a.yas3trigger.Run(); err != nil {
		return fmt.Errorf("yas3trigger run: %w", err)
	}

	return nil
}

func (a *App) Shutdown() error {
	if err := a.yas3trigger.Shutdown(); err != nil {
		return fmt.Errorf("yas3trigger shutdown: %w", err)
	}

	return nil
}
