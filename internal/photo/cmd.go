package photo

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

func cmd(ctx context.Context, prog string, args ...string) error {
	cmd := exec.CommandContext(ctx, prog, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run stderr=`%s`: %w", stderr.String(), err)
	}

	return nil
}
