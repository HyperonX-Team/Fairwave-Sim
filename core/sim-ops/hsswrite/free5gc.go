// free5GC HSS write-back.
//
// free5GC (5G SA) stores subscriber data in MongoDB, like Open5GS, but
// across the UDR's provisioned-data collections rather than a single
// `subscribers` collection. The document shapes below mirror exactly what
// free5GC's own webconsole and the community free5gc-populate tool write
// (TS 29.503 data model):
//
//	subscriptionData.authenticationData.authenticationSubscription  (Ki/OPc)
//	subscriptionData.provisionedData.amData                          (AMBR, NSSAI)
//	subscriptionData.provisionedData.smData                          (per-slice DNN config)
//	subscriptionData.provisionedData.smfSelectionSubscriptionData    (DNN selection)
//	policyData.ues.amData / policyData.ues.smData                    (policy context)
//	subscriptionData.identityData                                    (msisdn -> supi map)
//
// Like the Open5GS drivers, this runs `docker exec <container> mongosh`
// so the credentials stay on the node. Collection names contain dots, so
// the eval uses bracket notation: db["<collection>"].
package hsswrite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/HyperonX-Team/Fairwave-Sim/core/sim-ops/simprov"
)

// DriverFree5GC is the driver name for the free5GC mongosh write-back.
const DriverFree5GC = "free5gc"

// Free5GCDBURI is the free5GC subscriber database (UDR) URI used by the
// lab compose.
const Free5GCDBURI = "mongodb://localhost:27017/free5gc"

// free5GC MongoDB collections (UDR provisioned data).
const (
	f5cAuthSubsColl   = "subscriptionData.authenticationData.authenticationSubscription"
	f5cAmDataColl     = "subscriptionData.provisionedData.amData"
	f5cSmDataColl     = "subscriptionData.provisionedData.smData"
	f5cSmfSelColl     = "subscriptionData.provisionedData.smfSelectionSubscriptionData"
	f5cAmPolicyColl   = "policyData.ues.amData"
	f5cSmPolicyColl   = "policyData.ues.smData"
	f5cIdentityColl   = "subscriptionData.identityData"
	f5cDefaultSD      = "010203" // lab slice differentiator (hex)
	f5cDefaultDNN     = "internet"
	f5cDefaultAmbrStr = "1 Gbps" // free5GC bit-rate string when no policy cap
)

// Free5GC upserts the subscriber record set via mongosh in the mongo
// container. PlmnID must be MCC+MNC concatenated (e.g. "99999").
type Free5GC struct {
	Container string
	DBURI     string
	PlmnID    string
	SST       int
	SD        string // hex string, e.g. "010203"
	run       runner
}

// NewFree5GC builds the free5GC driver. container is the mongo container
// name; plmnID is MCC+MNC (used as servingPlmnId in every document); an
// empty plmnID defaults to the lab PLMN (999/99, "99999").
func NewFree5GC(container, plmnID string) *Free5GC {
	if plmnID == "" {
		plmnID = "99999"
	}
	return &Free5GC{
		Container: container,
		DBURI:     Free5GCDBURI,
		PlmnID:    plmnID,
		SST:       1,
		SD:        f5cDefaultSD,
		run:       execRunner,
	}
}

// Add implements Writer: it upserts the full free5GC subscriber record set.
func (f *Free5GC) Add(ctx context.Context, sub simprov.Subscriber) error {
	ueID := "imsi-" + sub.IMSI
	snssai := map[string]any{"sst": f.SST, "sd": f.SD}
	up, dn := ambrStr(sub.QoSULMbps), ambrStr(sub.QoSDLMbps)

	statements := []string{
		f5cUpsert(f5cAuthSubsColl, map[string]any{"ueId": ueID}, map[string]any{
			"ueId":                          ueID,
			"authenticationMethod":          "5G_AKA",
			"authenticationManagementField": sub.AMF,
			"encPermanentKey":               sub.Ki,
			"encOpcKey":                     sub.OPc,
			"sequenceNumber":                map[string]any{"sqnScheme": "GENERAL", "sqn": sub.SQN},
		}),
		f5cUpsert(f5cAmDataColl, map[string]any{"ueId": ueID, "servingPlmnId": f.PlmnID}, map[string]any{
			"ueId":          ueID,
			"servingPlmnId": f.PlmnID,
			"gpsis":         []string{"msisdn-" + sub.MSISDN},
			"nssai": map[string]any{
				"defaultSingleNssais": []any{snssai},
				"singleNssais":        []any{snssai},
			},
			"subscribedUeAmbr": map[string]any{"uplink": up, "downlink": dn},
		}),
		f5cUpsert(f5cSmDataColl, map[string]any{"ueId": ueID, "servingPlmnId": f.PlmnID, "singleNssai": snssai}, map[string]any{
			"ueId":          ueID,
			"servingPlmnId": f.PlmnID,
			"singleNssai":   snssai,
			"dnnConfigurations": map[string]any{f5cDefaultDNN: map[string]any{
				"pduSessionTypes": map[string]any{"defaultSessionType": "IPV4", "allowedSessionTypes": []any{"IPV4"}},
				"sscModes":        map[string]any{"defaultSscMode": "SSC_MODE_1", "allowedSscModes": []any{"SSC_MODE_1"}},
				"sessionAmbr":     map[string]any{"uplink": up, "downlink": dn},
				"5gQosProfile":    map[string]any{"5qi": 9, "arp": map[string]any{"priorityLevel": 8}, "priorityLevel": 8},
			}},
		}),
		f5cUpsert(f5cSmfSelColl, map[string]any{"ueId": ueID, "servingPlmnId": f.PlmnID}, map[string]any{
			"ueId":          ueID,
			"servingPlmnId": f.PlmnID,
			"subscribedSnssaiInfos": map[string]any{
				f5cSnssaiKey(f.SST, f.SD): map[string]any{"dnnInfos": []any{map[string]any{"dnn": f5cDefaultDNN}}},
			},
		}),
		f5cUpsert(f5cAmPolicyColl, map[string]any{"ueId": ueID}, map[string]any{
			"ueId":      ueID,
			"subscCats": []string{"free5gc"},
		}),
		f5cUpsert(f5cSmPolicyColl, map[string]any{"ueId": ueID}, map[string]any{
			"ueId": ueID,
			"smPolicySnssaiData": map[string]any{
				f5cSnssaiKey(f.SST, f.SD): map[string]any{
					"snssai": snssai,
					"smPolicyDnnData": map[string]any{
						f5cDefaultDNN: map[string]any{"dnn": f5cDefaultDNN},
					},
				},
			},
		}),
		f5cUpsert(f5cIdentityColl, map[string]any{"ueId": ueID}, map[string]any{
			"ueId": ueID,
			"gpsi": "msisdn-" + sub.MSISDN,
		}),
	}

	out, err := f.run(ctx, "docker", "exec", f.Container, "mongosh", "--quiet", f.DBURI, "--eval", strings.Join(statements, "\n"))
	if err != nil {
		return fmt.Errorf("hsswrite: free5gc add %s: %w: %s", sub.IMSI, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove implements Writer: it deletes the subscriber from every record
// collection (deleteMany is idempotent).
func (f *Free5GC) Remove(ctx context.Context, imsi string) error {
	ueID := "imsi-" + imsi
	var statements []string
	for _, coll := range []string{f5cAuthSubsColl, f5cAmDataColl, f5cSmDataColl, f5cSmfSelColl, f5cAmPolicyColl, f5cSmPolicyColl, f5cIdentityColl} {
		statements = append(statements, fmt.Sprintf(`db["%s"].deleteMany({ueId: %q})`, coll, ueID))
	}
	out, err := f.run(ctx, "docker", "exec", f.Container, "mongosh", "--quiet", f.DBURI, "--eval", strings.Join(statements, "\n"))
	if err != nil {
		return fmt.Errorf("hsswrite: free5gc remove %s: %w: %s", imsi, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ambrStr renders a Mbps cap as a free5GC bit-rate string; 0 means the
// lab default (matches the seed convention of "1 Gbps").
func ambrStr(mbps int) string {
	if mbps <= 0 {
		return f5cDefaultAmbrStr
	}
	return fmt.Sprintf("%d Mbps", mbps)
}

// f5cSnssaiKey renders the smfSelection/policy map key: SST as 2 digits
// followed by the hex SD (e.g. "01010203").
func f5cSnssaiKey(sst int, sd string) string {
	return fmt.Sprintf("%02d%s", sst, sd)
}

// f5cUpsert builds one mongosh upsert statement for a dotted collection
// name using bracket notation.
func f5cUpsert(coll string, filter, doc map[string]any) string {
	fj, _ := json.Marshal(filter)
	dj, _ := json.Marshal(doc)
	return fmt.Sprintf(`db["%s"].updateOne(%s, {$set: %s}, {upsert: true})`, coll, fj, dj)
}
