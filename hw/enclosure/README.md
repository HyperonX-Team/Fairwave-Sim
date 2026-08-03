# Fairwave 260 mm pizza-box enclosure

Low-profile horizontal case for the dev and community tiers: SBC + SDR +
power fit in ~60 mm of height with the radio coupled to the lid for
heat-spreading. Aluminum sheet/CNC for the dev tier; ABS/ASA 3D-print
variant for the community tier.

## Measurements (mm)

| Dimension | Value | Tolerance |
| --- | --- | --- |
| Width (front-to-back depth x) | 260 | ±0.5 |
| Depth (side-to-side y) | 260 | ±0.5 |
| Height (z, with lid) | 65 | ±1 |
| Lid height (removable) | 8 | ±0.5 |
| Chassis plate thickness | 2 | ±0.2 |
| Vent area, side panels (total) | 2× 60×40 | slots 3 mm wide |
| Standoff height (board-to-floor) | 10 | ±0.5 |
| Weight (empty, aluminum) | ~1.6 kg | — |

## Panel cutouts

| Panel | Cutouts |
| --- | --- |
| Rear | DC jack 2.1×5.5; USB-C (SDR, optional); RJ45 (Ethernet/PoE) |
| Front | 4× SMA (SDR RF + optional GPSDO in/out), 2× N-type (CBRS variant) |
| Left/Right | Vent slot banks; fan opening 40×40 (right side, intake) |
| Top lid | Thermal pad contact window (80×80 over the SDR) |

SMA/N panels are chassis-grounded at the cutout ring; keep the SDR USB
connection off the chassis ground loop (isolated USB cable is acceptable in
lab).

## Airflow

- Front-to-back: intake fan (right side, 40 mm) → SDR heat-spreader → rear
  exhaust slots. Two 40 mm fans for the dev tier, one for community.
- Fan profile: 5V/12V PWM, ~4–6 CFM per fan; the golden image pins the fan
  curve to SDR die temp via the agent's thermal hook.
- The enclosure is fanless-capable in lab mode at ≤ 8 W total board power
  (natural convection, aluminum chassis as sink).

## Thermals

| Load | Ambient | Airflow | Expected SDR die (B200mini) |
| --- | --- | --- | --- |
| Idle lab | 25 °C | none | ~45–50 °C |
| ZMQ lab full load | 25 °C | 1×40 mm | ~60 °C |
| RF full load (CBRS, outdoor box) | 45 °C | 2×40 mm + heatsink | ~75 °C (derate TX duty) |

Keep the SDR ≤ 85 °C junction: above that, reduce `tx_gain`/duty or the
agent sets the fan to 100%. See `hw/bom/` for the thermal pad part.

## Assembly notes

1. Mount the SDR on the lid's thermal window with 0.5 W/mK pad, standoffs 10 mm.
2. Route SMA pigtails before installing the SBC; leave service loop ≥ 30 mm.
3. CBRS variant: N-type feed with surge protector at the rear panel; filter
   sits between antenna port and SDR TX/RX split.
4. Torque SMA/N to spec (0.9–1.1 N·m); never bend panel-mount RF connectors.
