// Package euicc implements a software eUICC: the eUICC side of the
// SGP.22-shaped profile download (ES9+ client + the ES10b key-agreement and
// profile installation logic) entirely in Go.
//
// It exists so the full SM-DP+ <-> eUICC loop runs in CI without hardware.
// The same package is the reference for porting the logic to real eSIM
// modules (SGP.32) or for validating against a physical phone's LPA with
// the ASN.1 transport (docs/adr/0013-esim.md).
package euicc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/crypto"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
)

// InstalledProfile is a profile the eUICC has installed and (optionally)
// enabled.
type InstalledProfile struct {
	Profile     *profile.Profile
	Enabled     bool
	InstalledAt time.Time
}

// EUICC is a single software eUICC instance.
type EUICC struct {
	EID       string
	Profiles  map[string]*InstalledProfile // by ICCID
	Installed []string                     // install history (ICCIDs)
	Now       func() time.Time
}

// New creates a fresh eUICC with a new EID.
func New() (*EUICC, error) {
	eid, err := profile.NewEID()
	if err != nil {
		return nil, err
	}
	return &EUICC{
		EID:      eid,
		Profiles: make(map[string]*InstalledProfile),
		Now:      time.Now,
	}, nil
}

// Download runs the full ES9+ flow against an SM-DP+ at smdpURL:
// initiateAuthentication -> authenticateClient -> getBoundProfilePackage
// (verify MAC, decrypt, install) -> confirmOrder -> handleNotification.
// On any crypto or protocol failure the session is cancelled and the
// profile is not installed.
func (e *EUICC) Download(ctx context.Context, smdpURL, activationCode string, httpClient *http.Client) (*InstalledProfile, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	// ES10b key agreement: ephemeral eUICC keypair + challenge.
	ephemeral, err := crypto.ECDHKeyPair()
	if err != nil {
		return nil, err
	}
	challenge := make([]byte, 16)
	if _, err := rand.Read(challenge); err != nil {
		return nil, err
	}

	// 1. initiateAuthentication
	var ia InitiateAuthResp
	iaReq := map[string]string{
		"activation_code": activationCode,
		"eid":             e.EID,
		"euicc_challenge": hex.EncodeToString(challenge),
		"euicc_ekpb":      hex.EncodeToString(crypto.ECDHPublic(ephemeral.PublicKey())),
	}
	iaResp, status, err := postJSON(ctx, httpClient, smdpURL+"/es9plus/initiateAuthentication", iaReq, &ia)
	if err != nil {
		return nil, fmt.Errorf("euicc: initiateAuthentication: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("euicc: initiateAuthentication: %d: %s", status, iaResp)
	}

	// Derive the shared secret and session keys.
	serverEphemeral, err := hex.DecodeString(ia.ServerEphemeral)
	if err != nil {
		return nil, fmt.Errorf("euicc: bad server ephemeral: %w", err)
	}
	shared, err := crypto.ECDHShared(ephemeral, serverEphemeral)
	if err != nil {
		return nil, err
	}
	keys, err := crypto.DeriveSessionKeys(shared)
	if err != nil {
		return nil, err
	}

	// 2. authenticateClient: prove shared-secret knowledge.
	serverChallenge, err := hex.DecodeString(ia.ServerChallenge)
	if err != nil {
		return nil, fmt.Errorf("euicc: bad server challenge: %w", err)
	}
	proof, err := crypto.CMAC(keys.Enc[:], append(append([]byte{}, serverChallenge...), challenge...))
	if err != nil {
		return nil, err
	}
	var auth MetadataResp
	acReq := map[string]string{
		"transaction_id": ia.TransactionID,
		"auth_proof":     hex.EncodeToString(proof),
	}
	acResp, status, err := postJSON(ctx, httpClient, smdpURL+"/es9plus/authenticateClient", acReq, &auth)
	if err != nil {
		return nil, fmt.Errorf("euicc: authenticateClient: %w", err)
	}
	if status != http.StatusOK {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: authenticateClient: %d: %s", status, acResp)
	}

	// 3. getBoundProfilePackage: fetch, verify, decrypt, install.
	var bpp BPPResponse
	bppReq := map[string]string{"transaction_id": ia.TransactionID}
	bppResp, status, err := postJSON(ctx, httpClient, smdpURL+"/es9plus/getBoundProfilePackage", bppReq, &bpp)
	if err != nil {
		return nil, fmt.Errorf("euicc: getBoundProfilePackage: %w", err)
	}
	if status != http.StatusOK {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: getBoundProfilePackage: %d: %s", status, bppResp)
	}
	der, err := base64.StdEncoding.DecodeString(bpp.BPP)
	if err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: bpp base64: %w", err)
	}
	ep, err := profile.DecodeBPP(der)
	if err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: bpp: %w", err)
	}
	if ep.SeqCounter != bpp.SeqCounter {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: seq counter mismatch %d != %d", ep.SeqCounter, bpp.SeqCounter)
	}
	macInput, err := profile.EncodeMACInput(ep.SeqCounter, ep.Ciphertext)
	if err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, err
	}
	expectedMAC, err := crypto.CMAC(keys.Mac[:], macInput)
	if err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, err
	}
	if !crypto.VerifyMAC(expectedMAC, ep.MAC) {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: bpp MAC verification failed")
	}
	plaintext, err := crypto.Decrypt(keys.Dek[:], ep.Ciphertext)
	if err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: bpp decrypt: %w", err)
	}
	p, err := profile.UnmarshalPayload(plaintext)
	if err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, err
	}
	if err := p.Validate(); err != nil {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, err
	}
	if p.EID != "" && p.EID != e.EID {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: profile pinned to EID %s, this eUICC is %s", p.EID, e.EID)
	}
	if _, exists := e.Profiles[p.ICCID]; exists {
		_ = e.cancel(ctx, httpClient, smdpURL, ia.TransactionID)
		return nil, fmt.Errorf("euicc: ICCID %s already installed", p.ICCID)
	}

	// Install: only now does the profile exist on this eUICC.
	installed := &InstalledProfile{
		Profile:     p,
		Enabled:     true,
		InstalledAt: e.Now().UTC(),
	}
	e.Profiles[p.ICCID] = installed
	e.Installed = append(e.Installed, p.ICCID)

	// 4+5. confirmOrder + handleNotification.
	result := "success"
	if _, status, err := postJSON(ctx, httpClient, smdpURL+"/es9plus/confirmOrder",
		map[string]string{"transaction_id": ia.TransactionID, "result": result}, nil); err != nil || status != http.StatusOK {
		e.remove(p.ICCID)
		return nil, fmt.Errorf("euicc: confirmOrder: status %d err %v", status, err)
	}
	if _, status, err := postJSON(ctx, httpClient, smdpURL+"/es9plus/handleNotification",
		map[string]string{"transaction_id": ia.TransactionID, "notification": "install-success"}, nil); err != nil || status != http.StatusOK {
		e.remove(p.ICCID)
		return nil, fmt.Errorf("euicc: handleNotification: status %d err %v", status, err)
	}
	return installed, nil
}

func (e *EUICC) cancel(ctx context.Context, client *http.Client, smdpURL, transactionID string) error {
	_, _, err := postJSON(ctx, client, smdpURL+"/es9plus/cancelSession",
		map[string]string{"transaction_id": transactionID}, nil)
	return err
}

func (e *EUICC) remove(iccid string) {
	delete(e.Profiles, iccid)
	for i, id := range e.Installed {
		if id == iccid {
			e.Installed = append(e.Installed[:i], e.Installed[i+1:]...)
			break
		}
	}
}

// ListProfiles returns the installed profiles sorted by ICCID.
func (e *EUICC) ListProfiles() []*InstalledProfile {
	out := make([]*InstalledProfile, 0, len(e.Profiles))
	for _, p := range e.Profiles {
		out = append(out, p)
	}
	// deterministic order for tests and CLI output
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Profile.ICCID < out[j-1].Profile.ICCID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// EnableProfile enables a previously installed profile.
func (e *EUICC) EnableProfile(iccid string) error {
	p, ok := e.Profiles[iccid]
	if !ok {
		return fmt.Errorf("euicc: profile %s not installed", iccid)
	}
	p.Enabled = true
	return nil
}

// DisableProfile disables a profile without deleting it.
func (e *EUICC) DisableProfile(iccid string) error {
	p, ok := e.Profiles[iccid]
	if !ok {
		return fmt.Errorf("euicc: profile %s not installed", iccid)
	}
	p.Enabled = false
	return nil
}

// InitiateAuthResp mirrors the SM-DP+ initiateAuthentication response.
type InitiateAuthResp struct {
	TransactionID   string `json:"transaction_id"`
	SMDPID          string `json:"smdp_id"`
	ServerChallenge string `json:"server_challenge"`
	ServerEphemeral string `json:"server_ephemeral"`
}

// MetadataResp mirrors the authenticateClient response (profile metadata,
// never credentials).
type MetadataResp struct {
	ICCID               string    `json:"iccid"`
	ProfileName         string    `json:"profile_name"`
	ProfileOwner        string    `json:"profile_owner"`
	ServiceProviderName string    `json:"service_provider_name"`
	Class               string    `json:"class"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// BPPResponse mirrors the getBoundProfilePackage response.
type BPPResponse struct {
	SeqCounter int    `json:"seq_counter"`
	BPP        string `json:"bpp"`
}

func postJSON(ctx context.Context, client *http.Client, url string, body, out any) (string, int, error) {
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return "", 0, err
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, r)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	if resp.StatusCode == http.StatusOK && out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return string(respBody), resp.StatusCode, err
		}
	}
	return string(respBody), resp.StatusCode, nil
}
