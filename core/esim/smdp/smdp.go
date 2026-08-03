// Package smdp implements the Fairwave SM-DP+ (Subscription Manager - Data
// Preparation Plus) server core: the ES9+ download flow between an eUICC's
// LPA and the SM-DP+ that prepares and delivers eSIM carrier profiles
// (GSMA SGP.22).
//
// This is a LAB implementation of the flow shape. The wire messages are
// JSON (the SGP.22 ES9+ transport is ASN.1/DER over HTTPS) and the
// cryptographic details are the lab-defined subset from crypto and profile
// packages. Conformance against GSMA tooling and physical phones is tracked
// in docs/adr/0013-esim.md. The RF/spectrum gates of Fairwave still apply:
// a downloaded profile only gains network access through the normal
// (lab-gated) RAN path.
package smdp

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/activation"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/crypto"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
)

// SessionStatus is the download-session state machine step.
type SessionStatus string

const (
	StatusPending       SessionStatus = "pending"
	StatusAuthenticated SessionStatus = "authenticated"
	StatusConfirmed     SessionStatus = "confirmed"
	StatusCancelled     SessionStatus = "cancelled"
)

// Session is one profile-download exchange with a target eUICC.
// Ephemeral private keys and derived session keys live in memory only.
type Session struct {
	TransactionID  string
	ActivationCode string
	EID            string
	ICCID          string
	Status         SessionStatus
	SeqCounter     int

	EuiccChallenge  []byte
	EuiccEKPb       []byte
	ServerChallenge []byte
	ServerEphemeral *ecdh.PrivateKey
	Keys            crypto.SessionKeys

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store persists download sessions. Mirrors the repo's swappable-store
// pattern (ADR 0006): MemStore for tests and the lab binary; a file-backed
// implementation can be added without touching the flow logic.
type Store interface {
	CreateSession(s *Session) error
	GetSession(transactionID string) (*Session, error)
	UpdateSession(s *Session) error
	DeleteSession(transactionID string) error
}

// MemStore is an in-memory Store.
type MemStore struct {
	Sessions map[string]*Session
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{Sessions: make(map[string]*Session)}
}

// CreateSession implements Store.
func (m *MemStore) CreateSession(s *Session) error {
	m.Sessions[s.TransactionID] = s
	return nil
}

// GetSession implements Store.
func (m *MemStore) GetSession(transactionID string) (*Session, error) {
	s, ok := m.Sessions[transactionID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSession, transactionID)
	}
	return s, nil
}

// UpdateSession implements Store.
func (m *MemStore) UpdateSession(s *Session) error {
	m.Sessions[s.TransactionID] = s
	return nil
}

// DeleteSession implements Store.
func (m *MemStore) DeleteSession(transactionID string) error {
	delete(m.Sessions, transactionID)
	return nil
}

// ErrUnknownActivationCode means the activation code does not resolve to a
// provisioned profile.
var ErrUnknownActivationCode = errors.New("smdp: unknown activation code")

// ErrUnknownSession means the transaction id does not match an open session.
var ErrUnknownSession = errors.New("smdp: unknown session")

// ErrSessionState means the requested step is invalid for the session state.
var ErrSessionState = errors.New("smdp: invalid session state")

// ErrProofFailed means the client did not prove knowledge of the shared
// secret (or the EID does not match the session).
var ErrProofFailed = errors.New("smdp: client authentication failed")

// ErrBadRequest means a malformed message.
var ErrBadRequest = errors.New("smdp: bad request")

// ProfileSource resolves an activation code to the carrier profile to
// deliver. The server never stores secrets itself.
type ProfileSource func(activationCode string) (*profile.Profile, error)

// Server is the SM-DP+ core. It is pure state machine + crypto: TLS and
// serving are the caller's concern (Handler provides the HTTP mapping).
type Server struct {
	ID     string
	Store  Store
	Source ProfileSource
	Now    func() time.Time
}

// NewServer builds a server with the given identifier, store and profile
// source.
func NewServer(id string, store Store, source ProfileSource) *Server {
	return &Server{
		ID:     id,
		Store:  store,
		Source: source,
		Now:    time.Now,
	}
}

// InitiateAuthRequest is the first ES9+ message from the eUICC.
type InitiateAuthRequest struct {
	ActivationCode string `json:"activation_code"`
	EID            string `json:"eid"`
	EuiccChallenge string `json:"euicc_challenge"` // hex
	EuiccEKPb      string `json:"euicc_ekpb"`      // hex, P-256 uncompressed
}

// InitiateAuthResponse carries the server's challenge and ephemeral key.
type InitiateAuthResponse struct {
	TransactionID   string `json:"transaction_id"`
	SMDPID          string `json:"smdp_id"`
	ServerChallenge string `json:"server_challenge"` // hex
	ServerEphemeral string `json:"server_ephemeral"` // hex
}

// InitiateAuthentication opens a download session for the activation code.
func (s *Server) InitiateAuthentication(req *InitiateAuthRequest) (*InitiateAuthResponse, error) {
	if req == nil || req.ActivationCode == "" || req.EID == "" || req.EuiccChallenge == "" || req.EuiccEKPb == "" {
		return nil, fmt.Errorf("%w: activation_code, eid, euicc_challenge, euicc_ekpb required", ErrBadRequest)
	}
	if len(req.EID) != 32 || !profile.ValidLuhn(req.EID) {
		return nil, fmt.Errorf("%w: malformed EID %q", ErrBadRequest, req.EID)
	}
	token, err := normalizeActivationCode(req.ActivationCode)
	if err != nil {
		return nil, err
	}
	p, err := s.Source(token)
	if err != nil {
		return nil, err
	}
	if p.EID != "" && p.EID != req.EID {
		return nil, fmt.Errorf("%w: profile pinned to EID %s, not %s", ErrBadRequest, p.EID, req.EID)
	}
	challenge, err := randomBytes(16)
	if err != nil {
		return nil, err
	}
	ephemeral, err := crypto.ECDHKeyPair()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		TransactionID:   newTransactionID(),
		ActivationCode:  token,
		EID:             req.EID,
		ICCID:           p.ICCID,
		Status:          StatusPending,
		SeqCounter:      0,
		EuiccChallenge:  mustHex(req.EuiccChallenge),
		EuiccEKPb:       mustHex(req.EuiccEKPb),
		ServerChallenge: challenge,
		ServerEphemeral: ephemeral,
		CreatedAt:       s.Now().UTC(),
		UpdatedAt:       s.Now().UTC(),
	}
	if err := s.Store.CreateSession(sess); err != nil {
		return nil, err
	}
	return &InitiateAuthResponse{
		TransactionID:   sess.TransactionID,
		SMDPID:          s.ID,
		ServerChallenge: hex.EncodeToString(challenge),
		ServerEphemeral: hex.EncodeToString(crypto.ECDHPublic(ephemeral.PublicKey())),
	}, nil
}

// AuthClientRequest proves possession of the shared secret.
type AuthClientRequest struct {
	TransactionID string `json:"transaction_id"`
	AuthProof     string `json:"auth_proof"` // hex, CMAC over challenge pair
}

// AuthClientResponse carries the profile metadata (never the credentials).
type AuthClientResponse struct {
	ICCID               string    `json:"iccid"`
	ProfileName         string    `json:"profile_name"`
	ProfileOwner        string    `json:"profile_owner"`
	ServiceProviderName string    `json:"service_provider_name"`
	Class               string    `json:"class"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// AuthenticateClient verifies the client proof and derives session keys.
func (s *Server) AuthenticateClient(req *AuthClientRequest) (*AuthClientResponse, error) {
	if req == nil || req.TransactionID == "" || req.AuthProof == "" {
		return nil, fmt.Errorf("%w: transaction_id and auth_proof required", ErrBadRequest)
	}
	sess, err := s.Store.GetSession(req.TransactionID)
	if err != nil {
		return nil, err
	}
	if sess.Status != StatusPending {
		return nil, fmt.Errorf("%w: session is %s", ErrSessionState, sess.Status)
	}
	shared, err := crypto.ECDHShared(sess.ServerEphemeral, sess.EuiccEKPb)
	if err != nil {
		return nil, fmt.Errorf("%w: bad eckpb", ErrProofFailed)
	}
	keys, err := crypto.DeriveSessionKeys(shared)
	if err != nil {
		return nil, err
	}
	macInput := append(append([]byte{}, sess.ServerChallenge...), sess.EuiccChallenge...)
	expected, err := crypto.CMAC(keys.Enc[:], macInput)
	if err != nil {
		return nil, err
	}
	got, err := hex.DecodeString(req.AuthProof)
	if err != nil || !crypto.VerifyMAC(expected, got) {
		return nil, ErrProofFailed
	}
	sess.Keys = keys
	sess.Status = StatusAuthenticated
	sess.UpdatedAt = s.Now().UTC()
	if err := s.Store.UpdateSession(sess); err != nil {
		return nil, err
	}
	p, err := s.Source(sess.ActivationCode)
	if err != nil {
		return nil, err
	}
	return &AuthClientResponse{
		ICCID:               p.ICCID,
		ProfileName:         p.ProfileName,
		ProfileOwner:        p.ProfileOwner,
		ServiceProviderName: p.ServiceProviderName,
		Class:               p.Class,
		ExpiresAt:           p.ExpiresAt,
	}, nil
}

// BPPResponse is the encrypted bound profile package.
type BPPResponse struct {
	SeqCounter int    `json:"seq_counter"`
	BPP        string `json:"bpp"` // base64 DER
}

// GetBoundProfilePackage encrypts and MACs the profile for this session.
func (s *Server) GetBoundProfilePackage(transactionID string) (*BPPResponse, error) {
	sess, err := s.Store.GetSession(transactionID)
	if err != nil {
		return nil, err
	}
	if sess.Status != StatusAuthenticated && sess.Status != StatusConfirmed {
		return nil, fmt.Errorf("%w: session is %s", ErrSessionState, sess.Status)
	}
	p, err := s.Source(sess.ActivationCode)
	if err != nil {
		return nil, err
	}
	payload, err := p.MarshalPayload()
	if err != nil {
		return nil, err
	}
	ciphertext, err := crypto.Encrypt(sess.Keys.Dek[:], payload)
	if err != nil {
		return nil, err
	}
	sess.SeqCounter++
	macInput, err := profile.EncodeMACInput(sess.SeqCounter, ciphertext)
	if err != nil {
		return nil, err
	}
	mac, err := crypto.CMAC(sess.Keys.Mac[:], macInput)
	if err != nil {
		return nil, err
	}
	der, err := profile.EncodeBPP(profile.EncryptedProfile{
		SeqCounter: sess.SeqCounter,
		Ciphertext: ciphertext,
		MAC:        mac,
	})
	if err != nil {
		return nil, err
	}
	sess.UpdatedAt = s.Now().UTC()
	if err := s.Store.UpdateSession(sess); err != nil {
		return nil, err
	}
	return &BPPResponse{
		SeqCounter: sess.SeqCounter,
		BPP:        base64.StdEncoding.EncodeToString(der),
	}, nil
}

// ConfirmRequest finalizes the download.
type ConfirmRequest struct {
	TransactionID string `json:"transaction_id"`
	Result        string `json:"result"` // "success" | "failure"
}

// ConfirmOrder records the eUICC's install result.
func (s *Server) ConfirmOrder(req *ConfirmRequest) error {
	if req == nil || req.TransactionID == "" {
		return fmt.Errorf("%w: transaction_id required", ErrBadRequest)
	}
	if req.Result != "success" && req.Result != "failure" {
		return fmt.Errorf("%w: result must be success or failure", ErrBadRequest)
	}
	sess, err := s.Store.GetSession(req.TransactionID)
	if err != nil {
		return err
	}
	if sess.Status != StatusAuthenticated && sess.Status != StatusConfirmed {
		return fmt.Errorf("%w: session is %s", ErrSessionState, sess.Status)
	}
	sess.Status = StatusConfirmed
	sess.UpdatedAt = s.Now().UTC()
	return s.Store.UpdateSession(sess)
}

// NotificationRequest is a post-install notification to the SM-DP+.
type NotificationRequest struct {
	TransactionID string `json:"transaction_id"`
	Notification  string `json:"notification"` // e.g. "install-success"
}

// HandleNotification acknowledges an eUICC notification.
func (s *Server) HandleNotification(req *NotificationRequest) error {
	if req == nil || req.TransactionID == "" || req.Notification == "" {
		return fmt.Errorf("%w: transaction_id and notification required", ErrBadRequest)
	}
	sess, err := s.Store.GetSession(req.TransactionID)
	if err != nil {
		return err
	}
	if sess.Status == StatusCancelled {
		return fmt.Errorf("%w: session is cancelled", ErrSessionState)
	}
	sess.UpdatedAt = s.Now().UTC()
	return s.Store.UpdateSession(sess)
}

// CancelSession aborts a download session and releases the profile for
// another attempt.
func (s *Server) CancelSession(transactionID string) error {
	sess, err := s.Store.GetSession(transactionID)
	if err != nil {
		return err
	}
	if sess.Status == StatusCancelled {
		return fmt.Errorf("%w: session is cancelled", ErrSessionState)
	}
	sess.Status = StatusCancelled
	sess.UpdatedAt = s.Now().UTC()
	return s.Store.UpdateSession(sess)
}

// normalizeActivationCode accepts either the full SGP.22 activation-code
// string ("LPA:1$smdp$token") or a bare token, and returns the bare token
// the ProfileSource resolves.
func normalizeActivationCode(s string) (string, error) {
	if strings.HasPrefix(s, "LPA:1$") {
		c, err := activation.Parse(s)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrBadRequest, err)
		}
		return c.ActivationCode, nil
	}
	if activation.ValidToken(s) {
		return s, nil
	}
	return "", fmt.Errorf("%w: malformed activation code %q", ErrBadRequest, s)
}

func newTransactionID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("tx-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]any{"error": map[string]any{"code": "esim_error", "message": message}})
}

func httpStatusFor(err error) int {
	switch {
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnknownSession):
		return http.StatusNotFound
	case errors.Is(err, ErrSessionState):
		return http.StatusConflict
	case errors.Is(err, ErrProofFailed):
		return http.StatusUnauthorized
	case errors.Is(err, ErrUnknownActivationCode):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// Handler mounts the ES9+ endpoints as JSON over HTTPS (TLS is terminated
// by the caller, e.g. the control-plane daemon or an operator reverse
// proxy).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/es9plus/initiateAuthentication", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req InitiateAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		resp, err := s.InitiateAuthentication(&req)
		if err != nil {
			writeError(w, httpStatusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("/es9plus/authenticateClient", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req AuthClientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		resp, err := s.AuthenticateClient(&req)
		if err != nil {
			writeError(w, httpStatusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("/es9plus/getBoundProfilePackage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			TransactionID string `json:"transaction_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		resp, err := s.GetBoundProfilePackage(req.TransactionID)
		if err != nil {
			writeError(w, httpStatusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("/es9plus/confirmOrder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req ConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.ConfirmOrder(&req); err != nil {
			writeError(w, httpStatusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/es9plus/handleNotification", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req NotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.HandleNotification(&req); err != nil {
			writeError(w, httpStatusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/es9plus/cancelSession", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required")
			return
		}
		var req struct {
			TransactionID string `json:"transaction_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.CancelSession(req.TransactionID); err != nil {
			writeError(w, httpStatusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}
