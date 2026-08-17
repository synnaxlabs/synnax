// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

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
  label: "Cavity pump",
  Primitive: Cavity,
});
const centrifugal = createToggle({
  variant: "centrifugalCompressor",
  name: "Centrifugal compressor",
  Primitive: Centrifugal,
});
const compressor = createToggle({
  variant: "compressor",
  name: "Compressor",
  Primitive: Compressor,
});
const diaphragm = createToggle({
  variant: "diaphragmPump",
  name: "Diaphragm pump",
  Primitive: Diaphragm,
});
const ejection = createToggle({
  variant: "ejectionPump",
  name: "Ejection",
  label: "Ejection pump",
  Primitive: Ejection,
});
const ejector = createToggle({
  variant: "ejectorCompressor",
  name: "Ejector compressor",
  Primitive: Ejector,
});
const liquidRing = createToggle({
  variant: "liquidRingCompressor",
  name: "Liquid ring compressor",
  Primitive: LiquidRing,
});
const piston = createToggle({
  variant: "pistonPump",
  name: "Piston",
  label: "Piston pump",
  Primitive: Piston,
});
const pump = createToggle({ variant: "pump", name: "Pump", Primitive: Pump });
const rollerVane = createToggle({
  variant: "rollerVaneCompressor",
  name: "Roller vane compressor",
  Primitive: RollerVane,
});
const screw = createToggle({
  variant: "screwPump",
  name: "Screw",
  label: "Screw pump",
  Primitive: Screw,
});
const turbo = createToggle({
  variant: "turboCompressor",
  name: "Turbo compressor",
  Primitive: Turbo,
});
const vacuum = createToggle({
  variant: "vacuumPump",
  name: "Vacuum",
  label: "Vacuum pump",
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
