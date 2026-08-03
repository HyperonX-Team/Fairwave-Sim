#!/usr/bin/env bash
# Seed the Open5GS HSS with the two lab subscribers (dummy test vectors).
#
# Run inside the mongo container (compose mounts core/open5gs at /init):
#   docker compose -f deploy/docker-compose.yml exec mongo bash /init/hss-init.sh
#
# These are TEST-ONLY credentials from sim/test-vectors/lab-vectors.yaml.
# Never use, copy, or load them anywhere outside the lab. Real SIMs are
# minted offline by the provisioner (sim/provisioner).
set -euo pipefail

DB_URI="${HSS_DB_URI:-mongodb://localhost:27017/open5gs}"

mongosh --quiet "${DB_URI}" <<'EOF'
const subs = [
  {
    imsi: "999991234567001",
    k: "465B5CE8B199B49FAA5F0A2EE238A6BC",
    opc: "4D9B7A2C5E8F1A3B6C0D2E4F7A9B5C1D",
    msisdn: "9991234567001",
  },
  {
    imsi: "999991234567002",
    k: "8A1F3C9D5E7B2A4F6C8D0E1F3A5B7C9D",
    opc: "2E6B4A7D9F1C3E5B8A0D2F4C6E8B1A3F",
    msisdn: "9991234567002",
  },
];

let added = 0, updated = 0;
for (const s of subs) {
  const r = db.subscribers.updateOne(
    { imsi: s.imsi },
    {
      $set: {
        imsi: s.imsi,
        msisdn: [s.msisdn],
        access_restriction_data: 32,
        network_access_mode: 0,
        subscriber_status: 0,
        subscribed_rau_tau_timer: 12,
        // 3GPP AMF 0x8000, SQN starts at 0 (hex string as Open5GS stores it).
        security: {
          k: s.k,
          opc: s.opc,
          amf: "8000",
          sqn: "0000000000000000",
        },
        apn_list: [
          {
            apn: "internet",
            pcc_rule: [
              {
                qos: {
                  index: 9,
                  arp: {
                    priority_level: 8,
                    pre_emption_vulnerability: 1,
                    pre_emption_capability: 1,
                  },
                  mbr: { uplink: { value: 1000, unit: 8 }, downlink: { value: 1000, unit: 8 } },
                  gbr: { uplink: { value: 1000, unit: 8 }, downlink: { value: 1000, unit: 8 } },
                },
                flow: [
                  { direction: 2, description: "permit out ip from any to any" },
                  { direction: 1, description: "permit in ip from any to any" },
                ],
                name: "internet",
                type: 1,
              },
            ],
            qos: {
              class_id: 9,
              priority_level: 8,
              preemption_vulnerability: 1,
              preemption_capability: 1,
            },
            type: 0,
            ambr: {
              uplink: { value: 1, unit: 8 },
              downlink: { value: 1, unit: 8 },
            },
          },
        ],
        slice: [
          {
            sst: 1,
            default_indicator: true,
            session: [{ name: "internet", type: 3 }],
          },
        ],
        schema_version: 1,
        __v: 0,
      },
    },
    { upsert: true }
  );
  if (r.upsertedCount > 0) added++; else updated++;
}
print(`[hss-init] subscribers: ${added} inserted, ${updated} updated (imsi 999991234567001/002)`);
EOF

echo "[hss-init] done. Verify: docker compose exec open5gs sh -c 'open5gs-dbctl show 999991234567001'"
