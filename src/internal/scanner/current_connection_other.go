//go:build !darwin

package scanner

import (
	"context"
	"fmt"
	"runtime"
)

func CurrentConnection(context.Context) (Connection, error) {
	return Connection{}, fmt.Errorf("adding the current Wi-Fi connection from the dashboard is supported on macOS, not %s", runtime.GOOS)
}
