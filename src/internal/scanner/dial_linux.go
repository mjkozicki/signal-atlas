//go:build linux

package scanner

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func dialBound(ctx context.Context, iface, ip string, port int) (net.Conn, error) {
	i, e := net.InterfaceByName(iface)
	if e != nil {
		return nil, e
	}
	addrs, e := i.Addrs()
	if e != nil {
		return nil, e
	}
	target := net.ParseIP(ip)
	var source net.IP
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if ok && n.IP.To4() != nil && n.Contains(target) && !n.IP.Equal(target) {
			source = n.IP
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf("target is not in the connected interface's IPv4 subnet")
	}
	d := net.Dialer{Timeout: 2 * time.Second, LocalAddr: &net.TCPAddr{IP: source}, Control: func(_, _ string, raw syscall.RawConn) error {
		var bindErr error
		e := raw.Control(func(fd uintptr) {
			bindErr = unix.SetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_BINDTODEVICE, iface)
		})
		if e != nil {
			return e
		}
		return bindErr
	}}
	return d.DialContext(ctx, "tcp4", net.JoinHostPort(ip, strconv.Itoa(port)))
}
