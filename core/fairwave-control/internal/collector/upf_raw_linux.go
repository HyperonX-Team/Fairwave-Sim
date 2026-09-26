//go:build linux

package collector

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// RawSocket reads IP packets from a network interface with an AF_PACKET
// datagram socket (no link-layer header). Requires CAP_NET_RAW.
// Linux-only: see upf_raw_other.go for the non-Linux stub.
type RawSocket struct {
	fd   int
	ifi  int
	name string
}

// NewRawSocket binds an AF_PACKET socket to the named interface. The
// socket is SOCK_DGRAM so each read yields a full IP packet.
func NewRawSocket(iface string) (*RawSocket, error) {
	if iface == "" {
		return nil, fmt.Errorf("collector: upf iface required")
	}
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("collector: interface %q: %w", iface, err)
	}
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM, int(htons(unix.ETH_P_ALL)))
	if err != nil {
		return nil, fmt.Errorf("collector: AF_PACKET socket (CAP_NET_RAW required): %w", err)
	}
	sa := &unix.SockaddrLinklayer{Protocol: htons(unix.ETH_P_ALL), Ifindex: ifi.Index}
	if err := unix.Bind(fd, sa); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("collector: bind %s: %w", iface, err)
	}
	return &RawSocket{fd: fd, ifi: ifi.Index, name: iface}, nil
}

func (r *RawSocket) Name() string { return r.name }

// Next reads the next IP packet.
func (r *RawSocket) Next() ([]byte, error) {
	buf := make([]byte, 65536)
	for {
		n, _, err := unix.Recvfrom(r.fd, buf, 0)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			return buf[:n], nil
		}
	}
}

// Close releases the socket.
func (r *RawSocket) Close() error {
	return unix.Close(r.fd)
}

func htons(v uint16) uint16 {
	return v<<8 | v>>8
}
