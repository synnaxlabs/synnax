// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createToggle } from "@/schematic/node/common/create";
import { Cavity } from "@/schematic/node/pumps/Cavity";
import { Centrifugal } from "@/schematic/node/pumps/Centrifugal";
import { Compressor } from "@/schematic/node/pumps/Compressor";
import { Diaphragm } from "@/schematic/node/pumps/Diaphragm";
import { Ejection } from "@/schematic/node/pumps/Ejection";
import { Ejector } from "@/schematic/node/pumps/Ejector";
import { LiquidRing } from "@/schematic/node/pumps/LiquidRing";
import { Piston } from "@/schematic/node/pumps/Piston";
import { Pump } from "@/schematic/node/pumps/Pump";
import { RollerVane } from "@/schematic/node/pumps/RollerVane";
import { Screw } from "@/schematic/node/pumps/Screw";
import { Turbo } from "@/schematic/node/pumps/Turbo";
import { Vacuum } from "@/schematic/node/pumps/Vacuum";

const cavity = createToggle({
  variant: "cavityPump",
  name: "Cavity",
  label: "Cavity Pump",
  Primitive: Cavity,
});
const centrifugal = createToggle({
  variant: "centrifugalCompressor",
  name: "Centrifugal Compressor",
  Primitive: Centrifugal,
});
const compressor = createToggle({
  variant: "compressor",
  name: "Compressor",
  Primitive: Compressor,
});
const diaphragm = createToggle({
  variant: "diaphragmPump",
  name: "Diaphragm Pump",
  Primitive: Diaphragm,
});
const ejection = createToggle({
  variant: "ejectionPump",
  name: "Ejection",
  label: "Ejection Pump",
  Primitive: Ejection,
});
const ejector = createToggle({
  variant: "ejectorCompressor",
  name: "Ejector Compressor",
  Primitive: Ejector,
});
const liquidRing = createToggle({
  variant: "liquidRingCompressor",
  name: "Liquid Ring Compressor",
  Primitive: LiquidRing,
});
const piston = createToggle({
  variant: "pistonPump",
  name: "Piston",
  label: "Piston Pump",
  Primitive: Piston,
});
const pump = createToggle({ variant: "pump", name: "Pump", Primitive: Pump });
const rollerVane = createToggle({
  variant: "rollerVaneCompressor",
  name: "Roller Vane Compressor",
  Primitive: RollerVane,
});
const screw = createToggle({
  variant: "screwPump",
  name: "Screw",
  label: "Screw Pump",
  Primitive: Screw,
});
const turbo = createToggle({
  variant: "turboCompressor",
  name: "Turbo Compressor",
  Primitive: Turbo,
});
const vacuum = createToggle({
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
