// fairwave-hydra runs a Hydra data-plane node.
//
// It weaves the configured threads (independent SIMs, modems, and boxes)
// into one logical link: payloads arriving on the ingress socket are
// striped across the threads and reassembled in order at the peer's
// egress. Thread liveness is probed continuously, dead links are skipped,
// and live link health is reported back to the Fairwave control plane so
// the operator sees measured RTT and up/down state, not just declarations.
//
// Usage:
//
//	fairwave-hydra --config /etc/fairwave/hydra-edge.yaml
//
// See docs/architecture/hydra.md for the config schema and a worked
// two-box example.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/HyperonX-Team/Fairwave-Sim/core/hydra/node"
)

// Version is overridden at build time via -ldflags.
var Version = "0.1.0"

func main() {
	var configPath string
	var showVersion bool
	flag.StringVar(&configPath, "config", "", "path to a fairwave-hydra node config (YAML)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("fairwave-hydra %s\n", Version)
		return
	}
	if configPath == "" {
		log.Fatal("fairwave-hydra: --config is required (see docs/architecture/hydra.md)")
	}

	fc, err := node.LoadFile(configPath)
	if err != nil {
		log.Fatalf("fairwave-hydra: %v", err)
	}
	cfg, err := fc.Config()
	if err != nil {
		log.Fatalf("fairwave-hydra: %v", err)
	}
	cfg.Logf = func(format string, args ...any) { log.Printf(format, args...) }

	n, err := node.New(cfg)
	if err != nil {
		log.Fatalf("fairwave-hydra: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("fairwave-hydra %s: node=%s weave=%s anchor=%s threads=%d",
		Version, cfg.ID, cfg.WeaveID, cfg.Anchor, len(cfg.Threads))
	if addr := n.IngressAddr(); addr != "" {
		log.Printf("  ingress %s", addr)
	}
	for id, addr := range n.ThreadAddrs() {
		log.Printf("  thread %s bound %s", id, addr)
	}
	if fc.Report != "" {
		log.Printf("  reporting health to %s every %s", fc.Report, fc.ResolveReportInterval())
		go node.ReportLoop(ctx, n, fc.Report, fc.Token, fc.ResolveReportInterval(), cfg.Logf)
	}

	if err := n.Run(ctx); err != nil {
		log.Fatalf("fairwave-hydra: %v", err)
	}
	log.Printf("fairwave-hydra stopped cleanly")
}
