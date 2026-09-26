//go:build !linux

package collector

import "fmt"

// RawSocket is a non-Linux stub. The GTP-U AF_PACKET tap requires Linux +
// CAP_NET_RAW; on macOS/Windows the collector must use the CHF CDR or
// free5GC/AMF sources instead. Keeping this stub means `go build ./...`
// and `go vet ./...` stay green on every developer machine and in CI.
type RawSocket struct {
	name string
}

// NewRawSocket always fails on non-Linux with a clear, actionable error.
func NewRawSocket(iface string) (*RawSocket, error) {
	return nil, fmt.Errorf("collector: GTP-U tap requires Linux (AF_PACKET); interface %q unavailable on this OS — use collector source cdr or free5gc instead", iface)
}

func (r *RawSocket) Name() string { return r.name }

// Next always fails on non-Linux (NewRawSocket already fails first).
func (r *RawSocket) Next() ([]byte, error) {
	return nil, fmt.Errorf("collector: GTP-U tap requires Linux")
}

// Close is a no-op on non-Linux.
func (r *RawSocket) Close() error { return nil }
