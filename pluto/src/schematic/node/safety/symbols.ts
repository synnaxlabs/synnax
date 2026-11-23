// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { createStatic } from "@/schematic/node/common/create";
import { BurstDisc, SYMBOL as BURST_DISC } from "@/schematic/node/safety/BurstDisc";
import {
  FlameArrestor,
  SYMBOL as FLAME_ARRESTOR,
} from "@/schematic/node/safety/FlameArrestor";
import {
  FlameArrestorDetonation,
  SYMBOL as FLAME_ARRESTOR_DETONATION,
} from "@/schematic/node/safety/FlameArrestorDetonation";
import {
  FlameArrestorExplosion,
  SYMBOL as FLAME_ARRESTOR_EXPLOSION,
} from "@/schematic/node/safety/FlameArrestorExplosion";
import {
  FlameArrestorFireRes,
  SYMBOL as FLAME_ARRESTOR_FIRE_RES,
} from "@/schematic/node/safety/FlameArrestorFireRes";
import {
  FlameArrestorFireResDetonation,
  SYMBOL as FLAME_ARRESTOR_FIRE_RES_DETONATION,
} from "@/schematic/node/safety/FlameArrestorFireResDetonation";
import {
  IsoBurstDisc,
  SYMBOL as ISO_BURST_DISC,
} from "@/schematic/node/safety/IsoBurstDisc";

const burstDisc = createStatic({
  variant: "burst_disc",
  name: "Standard",
  label: "Burst disc",
  Primitive: BurstDisc,
});
const flameArrestor = createStatic({
  variant: "flame_arrestor",
  name: "Standard",
  label: "Flame arrestor",
  Primitive: FlameArrestor,
});
const flameArrestorDetonation = createStatic({
  variant: "flame_arrestor_detonation",
  name: "Detonation-proof",
  label: "Flame arrestor (detonation-proof)",
  Primitive: FlameArrestorDetonation,
});
const flameArrestorExplosion = createStatic({
  variant: "flame_arrestor_explosion",
  name: "Explosion-proof",
  label: "Flame arrestor (explosion-proof)",
  Primitive: FlameArrestorExplosion,
});
const flameArrestorFireRes = createStatic({
  variant: "flame_arrestor_fire_res",
  name: "Fire resistant",
  label: "Flame arrestor (fire resistant)",
  Primitive: FlameArrestorFireRes,
});
const flameArrestorFireResDetonation = createStatic({
  variant: "flame_arrestor_fire_res_detonation",
  name: "Fire resistant and detonation-proof",
  label: "Flame arrestor (fire resistant and detonation-proof)",
  Primitive: FlameArrestorFireResDetonation,
});
const isoBurstDisc = createStatic({
  variant: "iso_burst_disc",
  name: "ISO",
  label: "ISO burst disc",
  Primitive: IsoBurstDisc,
});

export const REGISTRY = {
  burst_disc: burstDisc.spec,
  flame_arrestor: flameArrestor.spec,
  flame_arrestor_detonation: flameArrestorDetonation.spec,
  flame_arrestor_explosion: flameArrestorExplosion.spec,
  flame_arrestor_fire_res: flameArrestorFireRes.spec,
  flame_arrestor_fire_res_detonation: flameArrestorFireResDetonation.spec,
  iso_burst_disc: isoBurstDisc.spec,
} as const;

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  burst_disc: BURST_DISC,
  flame_arrestor: FLAME_ARRESTOR,
  flame_arrestor_detonation: FLAME_ARRESTOR_DETONATION,
  flame_arrestor_explosion: FLAME_ARRESTOR_EXPLOSION,
  flame_arrestor_fire_res: FLAME_ARRESTOR_FIRE_RES,
  flame_arrestor_fire_res_detonation: FLAME_ARRESTOR_FIRE_RES_DETONATION,
  iso_burst_disc: ISO_BURST_DISC,
} as const satisfies Record<string, schematic.symbol.Spec>;
