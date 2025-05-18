package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tekig/photo-backup-server/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	a, err := app.New()
	if err != nil {
		return fmt.Errorf("new app: %w", err)
	}

	var chErr = make(chan error)
	go func() {
		defer close(chErr)
		if err := a.Run(); err != nil {
			chErr <- fmt.Errorf("run app: %w", err)
		}
	}()

	select {
	case err, ok := <-chErr:
		if !ok {
			err = fmt.Errorf("close app without error")
		}

		return err
	case <-ctx.Done():
	}

	if err := a.Shutdown(); err != nil {
		return fmt.Errorf("shutdown app: %w", err)
	}

	return nil
}
