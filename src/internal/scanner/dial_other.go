//go:build !linux

package scanner

import (
	"context"
	"fmt"
	"net"
)

func dialBound(context.Context, string, string, int) (net.Conn, error) {
	return nil, fmt.Errorf("interface-bound active checks are supported only on Linux")
}
