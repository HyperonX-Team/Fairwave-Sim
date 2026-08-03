package smdp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/crypto"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/euicc"
	"github.com/HyperonX-Team/Fairwave-Sim/core/esim/profile"
	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

const testActivationCode = "LABCODE1234"

func labProfile(t *testing.T, imsi string) *profile.Profile {
	t.Helper()
	sub, err := simprov.LoadTestVector(imsi)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profile.NewLabProfile(sub, "smdp.fairwave.test")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newTestServer(t *testing.T, store Store, imsi string) *httptest.Server {
	t.Helper()
	srv := NewServer("smdp.fairwave.test", store, func(ac string) (*profile.Profile, error) {
		if ac != testActivationCode {
			return nil, ErrUnknownActivationCode
		}
		return labProfile(t, imsi), nil
	})
	return httptest.NewServer(srv.Handler())
}

func firstSessionID(store *MemStore) string {
	for id := range store.Sessions {
		return id
	}
	return ""
}

func TestEndToEndDownload(t *testing.T) {
	sub, err := simprov.LoadTestVector("999991234567001")
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemStore()
	hs := newTestServer(t, store, sub.IMSI)
	defer hs.Close()

	e, err := euicc.New()
	if err != nil {
		t.Fatal(err)
	}

	installed, err := e.Download(context.Background(), hs.URL, "LPA:1$smdp.fairwave.test$"+testActivationCode, hs.Client())
	if err != nil {
		t.Fatal(err)
	}
	if installed.Profile.IMSI != sub.IMSI || installed.Profile.KI != sub.Ki {
		t.Fatal("installed profile must carry the subscriber credentials")
	}
	if !installed.Enabled {
		t.Fatal("fresh profile must be enabled")
	}
	if got := len(e.ListProfiles()); got != 1 {
		t.Fatalf("installed profiles = %d, want 1", got)
	}
	sess, err := store.GetSession(firstSessionID(store))
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != StatusConfirmed {
		t.Fatalf("session status = %s, want %s", sess.Status, StatusConfirmed)
	}
}

func TestEndToEndUnknownActivationCode(t *testing.T) {
	srv := NewServer("smdp.fairwave.test", NewMemStore(), func(string) (*profile.Profile, error) {
		return nil, ErrUnknownActivationCode
	})
	hs := httptest.NewServer(srv.Handler())
	defer hs.Close()

	e, _ := euicc.New()
	_, err := e.Download(context.Background(), hs.URL, "LPA:1$smdp.fairwave.test$WRONGCODE99", hs.Client())
	if err == nil {
		t.Fatal("expected failure for unknown activation code")
	}
	if got := len(e.ListProfiles()); got != 0 {
		t.Fatalf("profiles installed = %d, want 0", got)
	}
	if got := len(srv.Store.(*MemStore).Sessions); got != 0 {
		t.Fatalf("sessions created = %d, want 0", got)
	}
}

func TestEndToEndTamperedBPPRejected(t *testing.T) {
	// A server that corrupts the BPP ciphertext before responding must
	// cause the eUICC to refuse installation and cancel the session.
	sub, _ := simprov.LoadTestVector("999991234567001")
	inner := NewServer("smdp.fairwave.test", NewMemStore(), func(ac string) (*profile.Profile, error) {
		if ac != testActivationCode {
			return nil, ErrUnknownActivationCode
		}
		return labProfile(t, sub.IMSI), nil
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/es9plus/getBoundProfilePackage" {
			rr := httptest.NewRecorder()
			inner.Handler().ServeHTTP(rr, r)
			if rr.Code == http.StatusOK {
				var resp BPPResponse
				body := rr.Body.Bytes()
				if err := jsonDecode(body, &resp); err == nil {
					der, err := base64.StdEncoding.DecodeString(resp.BPP)
					if err == nil && len(der) > 8 {
						der[len(der)-8] ^= 0xFF
						resp.BPP = base64.StdEncoding.EncodeToString(der)
						body, _ = json.Marshal(resp)
					}
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
				return
			}
			w.WriteHeader(rr.Code)
			_, _ = w.Write(rr.Body.Bytes())
			return
		}
		inner.Handler().ServeHTTP(w, r)
	})
	hs := httptest.NewServer(handler)
	defer hs.Close()

	e, _ := euicc.New()
	_, err := e.Download(context.Background(), hs.URL, "LPA:1$smdp.fairwave.test$"+testActivationCode, hs.Client())
	if err == nil {
		t.Fatal("expected MAC verification failure on tampered BPP")
	}
	if got := len(e.ListProfiles()); got != 0 {
		t.Fatalf("profiles installed = %d, want 0", got)
	}
	store := inner.Store.(*MemStore)
	sess, err := store.GetSession(firstSessionID(store))
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != StatusCancelled {
		t.Fatalf("session status = %s, want %s", sess.Status, StatusCancelled)
	}
}

func TestSessionLifecycleAndKeyAgreement(t *testing.T) {
	sub, _ := simprov.LoadTestVector("999991234567002")
	store := NewMemStore()
	srv := NewServer("smdp.fairwave.test", store, func(ac string) (*profile.Profile, error) {
		if ac != testActivationCode {
			return nil, ErrUnknownActivationCode
		}
		return labProfile(t, sub.IMSI), nil
	})

	// eUICC side of the key agreement.
	ephem, err := crypto.ECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	challenge := bytes.Repeat([]byte{0x11}, 16)
	eid := labEID(t)

	ia, err := srv.InitiateAuthentication(&InitiateAuthRequest{
		ActivationCode: testActivationCode,
		EID:            eid,
		EuiccChallenge: hex.EncodeToString(challenge),
		EuiccEKPb:      hex.EncodeToString(crypto.ECDHPublic(ephem.PublicKey())),
	})
	if err != nil {
		t.Fatal(err)
	}
	sess, err := store.GetSession(ia.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != StatusPending {
		t.Fatalf("status = %s, want pending", sess.Status)
	}

	shared, err := crypto.ECDHShared(ephem, mustHex(ia.ServerEphemeral))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := crypto.DeriveSessionKeys(shared)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := crypto.CMAC(keys.Enc[:], append(append([]byte{}, mustHex(ia.ServerChallenge)...), challenge...))
	if err != nil {
		t.Fatal(err)
	}

	meta, err := srv.AuthenticateClient(&AuthClientRequest{
		TransactionID: ia.TransactionID,
		AuthProof:     hex.EncodeToString(proof),
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.ICCID == "" || meta.Class != "lab" {
		t.Fatalf("bad metadata: %+v", meta)
	}

	bpp, err := srv.GetBoundProfilePackage(ia.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	der, err := base64.StdEncoding.DecodeString(bpp.BPP)
	if err != nil {
		t.Fatal(err)
	}
	ep, err := profile.DecodeBPP(der)
	if err != nil {
		t.Fatal(err)
	}
	macInput, err := profile.EncodeMACInput(ep.SeqCounter, ep.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	mac, err := crypto.CMAC(keys.Mac[:], macInput)
	if err != nil {
		t.Fatal(err)
	}
	if !crypto.VerifyMAC(mac, ep.MAC) {
		t.Fatal("MAC mismatch with honestly derived keys")
	}
	plaintext, err := crypto.Decrypt(keys.Dek[:], ep.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profile.UnmarshalPayload(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if p.IMSI != sub.IMSI {
		t.Fatalf("profile IMSI = %s, want %s", p.IMSI, sub.IMSI)
	}

	// Wrong keys must fail both MAC and decryption.
	badKeys, _ := crypto.DeriveSessionKeys(bytes.Repeat([]byte{0x00}, 32))
	badMAC, _ := crypto.CMAC(badKeys.Mac[:], macInput)
	if crypto.VerifyMAC(badMAC, ep.MAC) {
		t.Fatal("MAC verified with wrong keys")
	}
	if _, err := crypto.Decrypt(badKeys.Dek[:], ep.Ciphertext); err == nil {
		t.Fatal("decrypt succeeded with wrong keys")
	}

	if err := srv.ConfirmOrder(&ConfirmRequest{TransactionID: ia.TransactionID, Result: "success"}); err != nil {
		t.Fatal(err)
	}
	if err := srv.ConfirmOrder(&ConfirmRequest{TransactionID: ia.TransactionID, Result: "success"}); err != nil {
		t.Fatal("second confirm must be idempotent")
	}
	if err := srv.HandleNotification(&NotificationRequest{TransactionID: ia.TransactionID, Notification: "install-success"}); err != nil {
		t.Fatal(err)
	}
	if err := srv.CancelSession(ia.TransactionID); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.GetBoundProfilePackage(ia.TransactionID); !errors.Is(err, ErrSessionState) {
		t.Fatalf("GetBPP on cancelled session: err = %v, want ErrSessionState", err)
	}
}

func TestInitiateRejectsBadInput(t *testing.T) {
	srv := NewServer("smdp.fairwave.test", NewMemStore(), func(ac string) (*profile.Profile, error) {
		if ac != testActivationCode {
			return nil, ErrUnknownActivationCode
		}
		return labProfile(t, "999991234567001"), nil
	})
	if _, err := srv.InitiateAuthentication(&InitiateAuthRequest{}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("empty request: err = %v, want ErrBadRequest", err)
	}
	if _, err := srv.InitiateAuthentication(&InitiateAuthRequest{
		ActivationCode: testActivationCode,
		EID:            "123", // not a valid EID
		EuiccChallenge: "00",
		EuiccEKPb:      "00",
	}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad EID: err = %v, want ErrBadRequest", err)
	}
	if _, err := srv.InitiateAuthentication(&InitiateAuthRequest{
		ActivationCode: "DEFINITELYUNKNOWN",
		EID:            labEID(t),
		EuiccChallenge: "00",
		EuiccEKPb:      "00",
	}); !errors.Is(err, ErrUnknownActivationCode) {
		t.Fatalf("unknown code: err = %v, want ErrUnknownActivationCode", err)
	}
	if _, err := srv.InitiateAuthentication(&InitiateAuthRequest{
		ActivationCode: "bad code with spaces",
		EID:            labEID(t),
		EuiccChallenge: "00",
		EuiccEKPb:      "00",
	}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("malformed code: err = %v, want ErrBadRequest", err)
	}
}

func labEID(t *testing.T) string {
	t.Helper()
	eid, err := profile.NewEID()
	if err != nil {
		t.Fatal(err)
	}
	return eid
}

func jsonDecode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
