package api

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/activation"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/registry"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/smdp"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/api"
	"github.com/HyperonX-Team/Fairwave-Sim/core/fairwave-control/config"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// esimSvc hosts the embedded lab SM-DP+ server and its activation-code
// registry. The registry file holds the encrypted profile payloads (0600,
// like the SIM vault); the SM-DP+ sessions themselves stay in memory.
type esimSvc struct {
	enabled bool
	reg     *registry.Registry
	smdp    *smdp.Server
	address string
	smdpID  string
	codeTTL time.Duration // 0 = no code-level expiry
	now     func() time.Time
}

// newEsimSvc builds the eSIM service from config. A nil option or a
// disabled one yields a disabled service whose routes answer 503.
func newEsimSvc(cfg *config.ControlConfig, o *ESIMOptions) *esimSvc {
	svc := &esimSvc{now: time.Now}
	if o == nil || !o.Enabled {
		return svc
	}
	path := o.RegistryPath
	if path == "" {
		path = filepath.Join(cfg.Server.DataDir, "esim", "registry.json")
	}
	reg, err := registry.Open(path)
	if err != nil {
		log.Printf("esim: registry %s: %v (eSIM disabled)", path, err)
		return svc
	}
	address := o.SMDPAddress
	if address == "" {
		address = "fairwave.local:8443"
	}
	smdpID := o.SMDPID
	if smdpID == "" {
		smdpID = "fairwave-esim"
	}
	svc.enabled = true
	svc.reg = reg
	svc.address = address
	svc.smdpID = smdpID
	// ProfileSource enforces the code-level policy: single-use codes refuse
	// a second download and codes expire after their TTL.
	resolve := func(token string) (*profile.Profile, error) {
		return reg.ResolvePolicy(token, o.SingleUse, svc.now().UTC())
	}
	srv := smdp.NewServer(smdpID, smdp.NewMemStore(), resolve)
	// Mark the code as downloaded as soon as the package is delivered.
	srv.OnDelivered = reg.MarkDownloaded
	svc.smdp = srv
	svc.codeTTL = o.CodeTTL
	return svc
}

// Handler returns the SM-DP+ ES9+ handler, or a 503 when disabled. The
// ES9+ surface does its own crypto auth and is mounted without the admin
// token (like /metrics): a phone's LPA must reach it.
func (e *esimSvc) Handler() http.Handler {
	if e == nil || !e.enabled {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeErr(w, http.StatusServiceUnavailable, "esim_disabled", "eSIM server is not configured")
		})
	}
	return e.smdp.Handler()
}

// issueEsim mints an eSIM profile for an existing SIM and registers its
// activation code. Lab-only by design (profile.NewLabProfile refuses
// non-lab classes): production profiles need the HSM/vault path.
func (s *Server) issueEsim(req *api.EsimIssueRequest, principal string) (*api.EsimIssueResponse, error) {
	if s.esim == nil || !s.esim.enabled {
		return nil, fmt.Errorf("eSIM server is not configured")
	}
	if req.IMSI == "" {
		return nil, fmt.Errorf("imsi required")
	}
	sim, ok := s.store.GetSIM(req.IMSI)
	if !ok {
		return nil, fmt.Errorf("no SIM %s in store; issue one first", req.IMSI)
	}
	if sim.Status == "revoked" || sim.Status == "suspended" || sim.Status == "expired" {
		return nil, fmt.Errorf("SIM %s is %s; reactivate before issuing an eSIM", req.IMSI, sim.Status)
	}
	// Credentials never cross the API; the lab server resolves them from the
	// reference test vectors (production: HSM import path, docs/sim-lifecycle).
	sub, err := simprov.LoadTestVector(req.IMSI)
	if err != nil {
		return nil, fmt.Errorf("eSIM issue: %v (lab vectors only; see docs/sim-lifecycle)", err)
	}
	token, err := activation.GenerateToken(12)
	if err != nil {
		return nil, err
	}
	var opts []profile.Option
	if req.EID != "" {
		opts = append(opts, profile.WithEID(req.EID))
	}
	p, err := profile.NewLabProfile(sub, s.esim.smdpID, opts...)
	if err != nil {
		return nil, err
	}
	var expiresAt time.Time
	if s.esim.codeTTL > 0 {
		expiresAt = s.esim.now().UTC().Add(s.esim.codeTTL)
	}
	if err := s.esim.reg.AddWithExpiry(token, p, expiresAt); err != nil {
		return nil, err
	}
	address := req.Address
	if address == "" {
		address = s.esim.address
	}
	code := activation.New(address, token)
	var qrB64 string
	if png, err := code.QR(); err == nil {
		qrB64 = base64.StdEncoding.EncodeToString(png)
	}
	detail := fmt.Sprintf("iccid=%s", p.ICCID)
	if req.EID != "" {
		detail += fmt.Sprintf(" eid=%s", req.EID)
	}
	s.audit(principal, "esim_issue", req.IMSI, detail)
	return &api.EsimIssueResponse{
		IMSI:           req.IMSI,
		ICCID:          p.ICCID,
		ActivationCode: token,
		SMDPAddress:    address,
		QRPNGBase64:    qrB64,
	}, nil
}

// listEsim returns the registered activation codes, newest first.
func (s *Server) listEsim() []api.EsimCode {
	if s.esim == nil || !s.esim.enabled {
		return nil
	}
	out := make([]api.EsimCode, 0, len(s.esim.reg.List()))
	for _, e := range s.esim.reg.List() {
		exp := e.Profile.ExpiresAt
		if e.ExpiresAt != nil {
			exp = *e.ExpiresAt
		}
		out = append(out, api.EsimCode{
			ActivationCode: e.Token,
			SMDPAddress:    s.esim.address,
			IMSI:           e.Profile.IMSI,
			ICCID:          e.Profile.ICCID,
			ProfileName:    e.Profile.ProfileName,
			EID:            e.Profile.EID,
			CreatedAt:      e.CreatedAt,
			DownloadedAt:   e.DownloadedAt,
			ExpiresAt:      exp,
		})
	}
	// newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// revokeEsim removes an activation code from the registry. Profiles already
// downloaded keep working on the eUICC; the code can no longer be scanned.
func (s *Server) revokeEsim(token, principal string) error {
	if s.esim == nil || !s.esim.enabled {
		return fmt.Errorf("eSIM server is not configured")
	}
	if token == "" {
		return fmt.Errorf("activation_code required")
	}
	if err := s.esim.reg.Revoke(token); err != nil {
		return err
	}
	s.audit(principal, "esim_revoke", token, "")
	return nil
}
