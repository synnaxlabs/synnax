// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { createToggle } from "@/schematic/node/common/create";
import { Cavity, SYMBOL as CAVITY } from "@/schematic/node/pumps/Cavity";
import { Centrifugal, SYMBOL as CENTRIFUGAL } from "@/schematic/node/pumps/Centrifugal";
import { Compressor, SYMBOL as COMPRESSOR } from "@/schematic/node/pumps/Compressor";
import { Diaphragm, SYMBOL as DIAPHRAGM } from "@/schematic/node/pumps/Diaphragm";
import { Ejection, SYMBOL as EJECTION } from "@/schematic/node/pumps/Ejection";
import { Ejector, SYMBOL as EJECTOR } from "@/schematic/node/pumps/Ejector";
import { LiquidRing, SYMBOL as LIQUID_RING } from "@/schematic/node/pumps/LiquidRing";
import { Piston, SYMBOL as PISTON } from "@/schematic/node/pumps/Piston";
import { Pump, SYMBOL as PUMP } from "@/schematic/node/pumps/Pump";
import { RollerVane, SYMBOL as ROLLER_VANE } from "@/schematic/node/pumps/RollerVane";
import { Screw, SYMBOL as SCREW } from "@/schematic/node/pumps/Screw";
import { SYMBOL as TURBO, Turbo } from "@/schematic/node/pumps/Turbo";
import { SYMBOL as VACUUM, Vacuum } from "@/schematic/node/pumps/Vacuum";

const cavity = createToggle({
  variant: "cavity_pump",
  name: "Cavity",
  label: "Cavity pump",
  Primitive: Cavity,
});
const centrifugal = createToggle({
  variant: "centrifugal_compressor",
  name: "Centrifugal compressor",
  Primitive: Centrifugal,
});
const compressor = createToggle({
  variant: "compressor",
  name: "Compressor",
  Primitive: Compressor,
});
const diaphragm = createToggle({
  variant: "diaphragm_pump",
  name: "Diaphragm pump",
  Primitive: Diaphragm,
});
const ejection = createToggle({
  variant: "ejection_pump",
  name: "Ejection",
  label: "Ejection pump",
  Primitive: Ejection,
});
const ejector = createToggle({
  variant: "ejector_compressor",
  name: "Ejector compressor",
  Primitive: Ejector,
});
const liquidRing = createToggle({
  variant: "liquid_ring_compressor",
  name: "Liquid ring compressor",
  Primitive: LiquidRing,
});
const piston = createToggle({
  variant: "piston_pump",
  name: "Piston",
  label: "Piston pump",
  Primitive: Piston,
});
const pump = createToggle({ variant: "pump", name: "Pump", Primitive: Pump });
const rollerVane = createToggle({
  variant: "roller_vane_compressor",
  name: "Roller vane compressor",
  Primitive: RollerVane,
});
const screw = createToggle({
  variant: "screw_pump",
  name: "Screw",
  label: "Screw pump",
  Primitive: Screw,
});
const turbo = createToggle({
  variant: "turbo_compressor",
  name: "Turbo compressor",
  Primitive: Turbo,
});
const vacuum = createToggle({
  variant: "vacuum_pump",
  name: "Vacuum",
  label: "Vacuum pump",
  Primitive: Vacuum,
});

export const REGISTRY = {
  cavity_pump: cavity.spec,
  centrifugal_compressor: centrifugal.spec,
  compressor: compressor.spec,
  diaphragm_pump: diaphragm.spec,
  ejection_pump: ejection.spec,
  ejector_compressor: ejector.spec,
  liquid_ring_compressor: liquidRing.spec,
  piston_pump: piston.spec,
  pump: pump.spec,
  roller_vane_compressor: rollerVane.spec,
  screw_pump: screw.spec,
  turbo_compressor: turbo.spec,
  vacuum_pump: vacuum.spec,
} as const;

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  cavity_pump: CAVITY,
  centrifugal_compressor: CENTRIFUGAL,
  compressor: COMPRESSOR,
  diaphragm_pump: DIAPHRAGM,
  ejection_pump: EJECTION,
  ejector_compressor: EJECTOR,
  liquid_ring_compressor: LIQUID_RING,
  piston_pump: PISTON,
  pump: PUMP,
  roller_vane_compressor: ROLLER_VANE,
  screw_pump: SCREW,
  turbo_compressor: TURBO,
  vacuum_pump: VACUUM,
} as const satisfies Record<string, schematic.symbol.Spec>;
