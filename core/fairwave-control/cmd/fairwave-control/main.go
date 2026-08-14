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

	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/config"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/internal/collector"
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

	var hss hsswrite.Writer
	if cfg.HSS.Driver == hsswrite.DriverFree5GC {
		// free5GC write-back: every document carries servingPlmnId, so the
		// real MCC+MNC must be supplied (NewFree5GC defaults to the lab PLMN).
		hss = hsswrite.NewFree5GC(cfg.HSS.Container, cfg.PLMN.MCC+cfg.PLMN.MNC)
	} else {
		hss = hsswrite.New(cfg.HSS.Driver, cfg.HSS.Container)
	}
	if _, ok := hss.(hsswrite.None); !ok {
		log.Printf("hss write-back enabled: driver=%s container=%s", cfg.HSS.Driver, cfg.HSS.Container)
	}

	var coll collector.Source = collector.None{}
	switch {
	case cfg.Core == "free5gc" && cfg.Collector.Enabled:
		// free5GC AMF OAM: live 5G registered-UE contexts straight from the
		// core (no infoAPI, no raw-socket tap). When a CHF CDR dir is
		// configured, the CDR meter (per-UE byte totals straight from the
		// charging function) runs alongside it - no GTP-U tap, no CAP_NET_RAW.
		amf := collector.NewFree5GC(collector.Free5GCConfig{
			AMFOAMURL: cfg.Free5GC.AMFOAMURL,
		})
		coll = amf
		if cfg.Free5GC.CDRDir != "" {
			coll = collector.Multi{amf, collector.NewCDR(collector.CDRConfig{Dir: cfg.Free5GC.CDRDir})}
			log.Printf("session collector enabled (free5gc): amf_oam=%s cdr_dir=%s", cfg.Free5GC.AMFOAMURL, cfg.Free5GC.CDRDir)
		} else {
			log.Printf("session collector enabled (free5gc): amf_oam=%s", cfg.Free5GC.AMFOAMURL)
		}
	case cfg.Collector.UPF.Enabled:
		// Per-UE GTP-U accounting tap: requires CAP_NET_RAW on the tap
		// interface and fails startup loudly if the interface is missing.
		u := collector.NewUPF(collector.UPFConfig{Iface: cfg.Collector.UPF.Iface})
		if _, err := u.Poll(context.Background()); err != nil {
			log.Fatalf("collector.upf: %v", err)
		}
		coll = u
		log.Printf("per-UE usage accounting enabled on %s (CAP_NET_RAW required)", cfg.Collector.UPF.Iface)
	case cfg.Collector.Enabled:
		coll = collector.NewOpen5GS(collector.Open5GSConfig{
			MMEURL: cfg.Collector.MMEURL,
			SMFURL: cfg.Collector.SMFURL,
		})
		log.Printf("session collector enabled: mme=%s smf=%s", cfg.Collector.MMEURL, cfg.Collector.SMFURL)
	}

	srv := api.NewWithOptions(cfg, st, id, policyAdapter{reg}, hss, api.Options{
		Collector: coll,
		ESIM: &api.ESIMOptions{
			Enabled:      cfg.ESIM.Enabled,
			RegistryPath: cfg.ESIM.RegistryPath,
			SMDPAddress:  cfg.ESIM.SMDPAddress,
			SMDPID:       cfg.ESIM.SMDPID,
			SingleUse:    cfg.ESIM.SingleUse,
			CodeTTL:      cfg.ESIM.CodeTTL,
		},
	})

	httpSrv := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
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

	// background jobs: session collection + SIM expiry sweeping
	go srv.RunBackground(ctx)
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
