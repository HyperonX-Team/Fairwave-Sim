// fairwave-control is the Fairwave control-plane daemon: identity,
// lifecycle, northbound REST API, and (in later milestones) southbound
// supervision of Open5GS and srsRAN.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/config"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/identity"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/store"
	"github.com/HyperonX-Team/Fairwave-Sim/core/policy"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/hsswrite"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to fairwave-control.yaml")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	id, err := identity.LoadOrCreate(cfg.Server.DataDir + "/identity")
	if err != nil {
		log.Fatalf("identity: %v", err)
	}

	st, err := store.Open(cfg.Server.DataDir + "/state")
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	reg, err := policy.DefaultRegistry(cfg.Spectrum.OverlayPath)
	if err != nil {
		log.Fatalf("spectrum: %v", err)
	}

	hss := hsswrite.New(cfg.HSS.Driver, cfg.HSS.Container)
	if _, ok := hss.(hsswrite.None); !ok {
		log.Printf("hss write-back enabled: driver=%s container=%s", cfg.HSS.Driver, cfg.HSS.Container)
	}

	srv := api.New(cfg, st, id, policyAdapter{reg}, hss)

	httpSrv := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("fairwave-control %s listening on %s (node %s, mode %s, country %s)",
			api.Version, cfg.Server.Listen, id.ID, cfg.Server.Mode, cfg.Server.Country)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	// graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.Shutdown)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Printf("fairwave-control stopped cleanly")
}

// policyAdapter adapts core/policy.Registry to the api.SpectrumChecker
// interface without importing policy into the api package.
type policyAdapter struct{ reg *policy.Registry }

func (a policyAdapter) Check(country, band string, indoor bool, licenseRef string) api.Verdict {
	v := a.reg.Check(country, band, indoor, licenseRef)
	return api.Verdict{Allowed: v.Allowed, Reasons: v.Reasons}
}
