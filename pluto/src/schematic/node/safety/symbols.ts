// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import type { Names } from "@/schematic/node/builtIns";
import { SYMBOL as BURST_DISC } from "@/schematic/node/safety/BurstDisc";
import { SYMBOL as FLAME_ARRESTOR } from "@/schematic/node/safety/FlameArrestor";
import { SYMBOL as FLAME_ARRESTOR_DETONATION } from "@/schematic/node/safety/FlameArrestorDetonation";
import { SYMBOL as FLAME_ARRESTOR_EXPLOSION } from "@/schematic/node/safety/FlameArrestorExplosion";
import { SYMBOL as FLAME_ARRESTOR_FIRE_RES } from "@/schematic/node/safety/FlameArrestorFireRes";
import { SYMBOL as FLAME_ARRESTOR_FIRE_RES_DETONATION } from "@/schematic/node/safety/FlameArrestorFireResDetonation";
import { SYMBOL as ISO_BURST_DISC } from "@/schematic/node/safety/IsoBurstDisc";

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

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  burst_disc: { name: "Standard", label: "Burst disc" },
  flame_arrestor: { name: "Standard", label: "Flame arrestor" },
  flame_arrestor_detonation: {
    name: "Detonation-proof",
    label: "Flame arrestor (detonation-proof)",
  },
  flame_arrestor_explosion: {
    name: "Explosion-proof",
    label: "Flame arrestor (explosion-proof)",
  },
  flame_arrestor_fire_res: {
    name: "Fire resistant",
    label: "Flame arrestor (fire resistant)",
  },
  flame_arrestor_fire_res_detonation: {
    name: "Fire resistant and detonation-proof",
    label: "Flame arrestor (fire resistant and detonation-proof)",
  },
  iso_burst_disc: { name: "ISO", label: "ISO burst disc" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
