// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { defineStatic } from "@/schematic/node/define";
import { BurstDisc } from "@/schematic/node/safety/BurstDisc";
import { FlameArrestor } from "@/schematic/node/safety/FlameArrestor";
import { FlameArrestorDetonation } from "@/schematic/node/safety/FlameArrestorDetonation";
import { FlameArrestorExplosion } from "@/schematic/node/safety/FlameArrestorExplosion";
import { FlameArrestorFireRes } from "@/schematic/node/safety/FlameArrestorFireRes";
import { FlameArrestorFireResDetonation } from "@/schematic/node/safety/FlameArrestorFireResDetonation";
import { IsoBurstDisc } from "@/schematic/node/safety/IsoBurstDisc";

const burstDisc = defineStatic({
  variant: "burstDisc",
  name: "Standard",
  label: "Burst Disc",
  Primitive: BurstDisc,
});
const flameArrestor = defineStatic({
  variant: "flameArrestor",
  name: "Standard",
  label: "Flame Arrestor",
  Primitive: FlameArrestor,
});
const flameArrestorDetonation = defineStatic({
  variant: "flameArrestorDetonation",
  name: "Detonation-Proof",
  label: "Flame Arrestor (Detonation-Proof)",
  Primitive: FlameArrestorDetonation,
});
const flameArrestorExplosion = defineStatic({
  variant: "flameArrestorExplosion",
  name: "Explosion-Proof",
  label: "Flame Arrestor (Explosion-Proof)",
  Primitive: FlameArrestorExplosion,
});
const flameArrestorFireRes = defineStatic({
  variant: "flameArrestorFireRes",
  name: "Fire Resistant",
  label: "Flame Arrestor (Fire Resistant)",
  Primitive: FlameArrestorFireRes,
});
const flameArrestorFireResDetonation = defineStatic({
  variant: "flameArrestorFireResDetonation",
  name: "Fire Resistant & Detonation-Proof",
  label: "Flame Arrestor (Fire Resistant and Detonation-Proof)",
  Primitive: FlameArrestorFireResDetonation,
});
const isoBurstDisc = defineStatic({
  variant: "isoBurstDisc",
  name: "ISO",
  label: "ISO Burst Disc",
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
