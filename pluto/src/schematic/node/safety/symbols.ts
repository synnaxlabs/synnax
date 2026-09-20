// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { createStatic } from "@/schematic/node/common/create";
import { BurstDisc } from "@/schematic/node/safety/BurstDisc";
import { FlameArrestor } from "@/schematic/node/safety/FlameArrestor";
import { FlameArrestorDetonation } from "@/schematic/node/safety/FlameArrestorDetonation";
import { FlameArrestorExplosion } from "@/schematic/node/safety/FlameArrestorExplosion";
import { FlameArrestorFireRes } from "@/schematic/node/safety/FlameArrestorFireRes";
import { FlameArrestorFireResDetonation } from "@/schematic/node/safety/FlameArrestorFireResDetonation";
import { IsoBurstDisc } from "@/schematic/node/safety/IsoBurstDisc";

const burstDisc = createStatic({
  variant: "burstDisc",
  name: "Standard",
  label: "Burst disc",
  Primitive: BurstDisc,
});
const flameArrestor = createStatic({
  variant: "flameArrestor",
  name: "Standard",
  label: "Flame arrestor",
  Primitive: FlameArrestor,
});
const flameArrestorDetonation = createStatic({
  variant: "flameArrestorDetonation",
  name: "Detonation-proof",
  label: "Flame arrestor (detonation-proof)",
  Primitive: FlameArrestorDetonation,
});
const flameArrestorExplosion = createStatic({
  variant: "flameArrestorExplosion",
  name: "Explosion-proof",
  label: "Flame arrestor (explosion-proof)",
  Primitive: FlameArrestorExplosion,
});
const flameArrestorFireRes = createStatic({
  variant: "flameArrestorFireRes",
  name: "Fire resistant",
  label: "Flame arrestor (fire resistant)",
  Primitive: FlameArrestorFireRes,
});
const flameArrestorFireResDetonation = createStatic({
  variant: "flameArrestorFireResDetonation",
  name: "Fire resistant and detonation-proof",
  label: "Flame arrestor (fire resistant and detonation-proof)",
  Primitive: FlameArrestorFireResDetonation,
});
const isoBurstDisc = createStatic({
  variant: "isoBurstDisc",
  name: "ISO",
  label: "ISO burst disc",
  Primitive: IsoBurstDisc,
});

export const REGISTRY = {
  burstDisc: burstDisc.spec,
  flameArrestor: flameArrestor.spec,
  flameArrestorDetonation: flameArrestorDetonation.spec,
  flameArrestorExplosion: flameArrestorExplosion.spec,
  flameArrestorFireRes: flameArrestorFireRes.spec,
  flameArrestorFireResDetonation: flameArrestorFireResDetonation.spec,
  isoBurstDisc: isoBurstDisc.spec,
} as const;

export const configZ = z.discriminatedUnion("variant", [
  burstDisc.configZ,
  flameArrestor.configZ,
  flameArrestorDetonation.configZ,
  flameArrestorExplosion.configZ,
  flameArrestorFireRes.configZ,
  flameArrestorFireResDetonation.configZ,
  isoBurstDisc.configZ,
]);
