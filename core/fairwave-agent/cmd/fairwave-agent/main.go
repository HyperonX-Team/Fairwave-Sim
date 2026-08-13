// fairwave-agent is the on-box health/watchdog daemon.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-agent/internal/agent"
)

func main() {
	var (
		controlURL string
		nodeID     string
		token      string
		interval   time.Duration
		dataDir    string
		enableRF   bool
	)
	flag.StringVar(&controlURL, "control", "http://localhost:8080", "control plane URL")
	flag.StringVar(&nodeID, "node-id", "", "node id (default: hostname)")
	flag.StringVar(&token, "token", os.Getenv("FAIRWAVE_ADMIN_TOKEN"), "bearer token for /v1/telemetry")
	flag.DurationVar(&interval, "interval", 10*time.Second, "heartbeat interval")
	flag.StringVar(&dataDir, "data-dir", "./data", "data dir for watchdog state")
	flag.BoolVar(&enableRF, "enable-rf", false, "MUST NOT be enabled without tx-arm approval (stays false)")
	flag.Parse()

	if nodeID == "" {
		nodeID, _ = os.Hostname()
	}

	// safety: RF stays off unless explicitly armed; even then, lab default
	// keeps the radio in zmq mode. See docs/spectrum-and-law/.
	if enableRF {
		log.Printf("WARNING: --enable-rf set; radio is only allowed after spectrum gate approval")
	}

	a := agent.New(agent.Config{
		ControlURL: controlURL,
		NodeID:     nodeID,
		Token:      token,
		Interval:   interval,
		DataDir:    dataDir,
		EnableRF:   enableRF,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// kick watchdog every tick alongside heartbeats
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			if err := a.TouchWatchdog(); err != nil {
				log.Printf("watchdog: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	log.Printf("fairwave-agent (%s) reporting to %s every %s (rf=%v)",
		nodeID, controlURL, interval, enableRF)
	if err := a.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
