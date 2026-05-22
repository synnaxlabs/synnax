// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { defineToggle } from "@/schematic/node/define";
import { Primitive as Cavity } from "@/schematic/node/pumps/Cavity";
import { Primitive as Centrifugal } from "@/schematic/node/pumps/Centrifugal";
import { Primitive as Compressor } from "@/schematic/node/pumps/Compressor";
import { Primitive as Diaphragm } from "@/schematic/node/pumps/Diaphragm";
import { Primitive as Ejection } from "@/schematic/node/pumps/Ejection";
import { Primitive as Ejector } from "@/schematic/node/pumps/Ejector";
import { Primitive as LiquidRing } from "@/schematic/node/pumps/LiquidRing";
import { Primitive as Piston } from "@/schematic/node/pumps/Piston";
import { Primitive as Pump } from "@/schematic/node/pumps/Pump";
import { Primitive as RollerVane } from "@/schematic/node/pumps/RollerVane";
import { Primitive as Screw } from "@/schematic/node/pumps/Screw";
import { Primitive as Turbo } from "@/schematic/node/pumps/Turbo";
import { Primitive as Vacuum } from "@/schematic/node/pumps/Vacuum";

const cavity = defineToggle({
  variant: "cavityPump",
  name: "Cavity",
  label: "Cavity Pump",
  Primitive: Cavity,
});
const centrifugal = defineToggle({
  variant: "centrifugalCompressor",
  name: "Centrifugal Compressor",
  label: "Centrifugal Compressor",
  Primitive: Centrifugal,
});
const compressor = defineToggle({
  variant: "compressor",
  name: "Compressor",
  label: "Compressor",
  Primitive: Compressor,
});
const diaphragm = defineToggle({
  variant: "diaphragmPump",
  name: "Diaphragm Pump",
  label: "Diaphragm Pump",
  Primitive: Diaphragm,
});
const ejection = defineToggle({
  variant: "ejectionPump",
  name: "Ejection",
  label: "Ejection Pump",
  Primitive: Ejection,
});
const ejector = defineToggle({
  variant: "ejectorCompressor",
  name: "Ejector Compressor",
  label: "Ejector Compressor",
  Primitive: Ejector,
});
const liquidRing = defineToggle({
  variant: "liquidRingCompressor",
  name: "Liquid Ring Compressor",
  label: "Liquid Ring Compressor",
  Primitive: LiquidRing,
});
const piston = defineToggle({
  variant: "pistonPump",
  name: "Piston",
  label: "Piston Pump",
  Primitive: Piston,
});
const pump = defineToggle({
  variant: "pump",
  name: "Pump",
  label: "Pump",
  Primitive: Pump,
});
const rollerVane = defineToggle({
  variant: "rollerVaneCompressor",
  name: "Roller Vane Compressor",
  label: "Roller Vane Compressor",
  Primitive: RollerVane,
});
const screw = defineToggle({
  variant: "screwPump",
  name: "Screw",
  label: "Screw Pump",
  Primitive: Screw,
});
const turbo = defineToggle({
  variant: "turboCompressor",
  name: "Turbo Compressor",
  label: "Turbo Compressor",
  Primitive: Turbo,
});
const vacuum = defineToggle({
  variant: "vacuumPump",
  name: "Vacuum",
  label: "Vacuum Pump",
  Primitive: Vacuum,
});

export const REGISTRY = {
  cavityPump: cavity.spec,
  centrifugalCompressor: centrifugal.spec,
  compressor: compressor.spec,
  diaphragmPump: diaphragm.spec,
  ejectionPump: ejection.spec,
  ejectorCompressor: ejector.spec,
  liquidRingCompressor: liquidRing.spec,
  pistonPump: piston.spec,
  pump: pump.spec,
  rollerVaneCompressor: rollerVane.spec,
  screwPump: screw.spec,
  turboCompressor: turbo.spec,
  vacuumPump: vacuum.spec,
} as const;

export const configZ = z.discriminatedUnion("variant", [
  cavity.configZ,
  centrifugal.configZ,
  compressor.configZ,
  diaphragm.configZ,
  ejection.configZ,
  ejector.configZ,
  liquidRing.configZ,
  piston.configZ,
  pump.configZ,
  rollerVane.configZ,
  screw.configZ,
  turbo.configZ,
  vacuum.configZ,
]);
