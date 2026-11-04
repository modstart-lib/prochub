//go:build !linux && !darwin && !windows

package updater

import (
	"context"
	"fmt"
	"runtime"
)

func installPlatform(ctx context.Context, asset Asset, downloaded string, opts InstallOptions) (InstallResult, error) {
	return InstallResult{}, fmt.Errorf("%w: %s", ErrUnsupported, runtime.GOOS)
}
