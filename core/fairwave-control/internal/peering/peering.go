// Package peering implements the neighborhood mesh MVP: mDNS discovery of
// neighboring boxes (zeroconf), plus WireGuard config generation. Full
// route exchange lands in M2; this package establishes the interface.
package peering

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/grandcat/zeroconf"
)

// ServiceType is the mDNS service advertised by Fairwave boxes.
const ServiceType = "_fairwave._tcp"

// ServiceName is the instance name every box advertises.
const ServiceName = "fairwave"

// Discoverer runs mDNS browse and hands found peers to a callback.
type Discoverer struct {
	ServiceType string
	Interval    time.Duration
	OnPeer      func(peer api.Peer)
}

// NewDiscoverer creates a discoverer with sane defaults.
func NewDiscoverer(onPeer func(peer api.Peer)) *Discoverer {
	return &Discoverer{ServiceType: ServiceType, Interval: 30 * time.Second, OnPeer: onPeer}
}

// Run starts browsing; blocks until ctx is done.
func (d *Discoverer) Run(ctx context.Context, advertise bool, port int) error {
	if advertise {
		srv, err := zeroconf.Register(ServiceName, ServiceType, "local.", port, []string{"fairwave=1"}, nil)
		if err != nil {
			return fmt.Errorf("mDNS register: %w", err)
		}
		defer srv.Shutdown()
		log.Printf("peering: advertising %s on udp/%d", ServiceType, port)
	}
	t := time.NewTicker(d.Interval)
	defer t.Stop()
	for {
		if err := d.browseOnce(ctx); err != nil {
			log.Printf("peering: browse: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (d *Discoverer) browseOnce(ctx context.Context) error {
	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		for e := range entries {
			if len(e.AddrIPv4) == 0 {
				continue
			}
			peer := api.Peer{
				Name:     e.Instance,
				Endpoint: net.JoinHostPort(e.AddrIPv4[0].String(), strconv.Itoa(e.Port)),
				Status:   "up",
				LastSeen: time.Now(),
			}
			for _, txt := range e.Text {
				if len(txt) > 8 && txt[:8] == "pubkey=" {
					peer.PubKey = txt[8:]
				}
			}
			if d.OnPeer != nil {
				d.OnPeer(peer)
			}
		}
	}()
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return err
	}
	err = resolver.Browse(ctx, d.ServiceType, "local.", entries)
	if ctx.Err() != nil {
		return nil
	}
	return err
}

// ---- WireGuard config generation (M2 data plane) ----

// WGConfig renders a wg-quick compatible config for one node.
type WGConfig struct {
	Iface      string
	ListenPort int
	PrivKey    string // NEVER log; operator pastes via env/vault
	Addresses  []string
	Peers      []WgPeer
}

// WgPeer is one remote endpoint.
type WgPeer struct {
	PubKey              string
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int // 25 typical for NAT
}

// Render returns the wg-quick config text.
func (w WGConfig) Render() string {
	out := fmt.Sprintf("[Interface]\nPrivateKey = %s\nListenPort = %d\n", w.PrivKey, w.ListenPort)
	for _, a := range w.Addresses {
		out += fmt.Sprintf("Address = %s\n", a)
	}
	for _, p := range w.Peers {
		out += "\n[Peer]\n"
		out += fmt.Sprintf("PublicKey = %s\n", p.PubKey)
		if p.Endpoint != "" {
			out += fmt.Sprintf("Endpoint = %s\n", p.Endpoint)
		}
		if len(p.AllowedIPs) > 0 {
			ips := ""
			for i, ip := range p.AllowedIPs {
				if i > 0 {
					ips += ", "
				}
				ips += ip
			}
			out += fmt.Sprintf("AllowedIPs = %s\n", ips)
		}
		if p.PersistentKeepalive > 0 {
			out += fmt.Sprintf("PersistentKeepalive = %d\n", p.PersistentKeepalive)
		}
	}
	return out
}
