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
import { SYMBOL as CAVITY } from "@/schematic/node/pumps/Cavity";
import { SYMBOL as CENTRIFUGAL } from "@/schematic/node/pumps/Centrifugal";
import { SYMBOL as COMPRESSOR } from "@/schematic/node/pumps/Compressor";
import { SYMBOL as DIAPHRAGM } from "@/schematic/node/pumps/Diaphragm";
import { SYMBOL as EJECTION } from "@/schematic/node/pumps/Ejection";
import { SYMBOL as EJECTOR } from "@/schematic/node/pumps/Ejector";
import { SYMBOL as LIQUID_RING } from "@/schematic/node/pumps/LiquidRing";
import { SYMBOL as PISTON } from "@/schematic/node/pumps/Piston";
import { SYMBOL as PUMP } from "@/schematic/node/pumps/Pump";
import { SYMBOL as ROLLER_VANE } from "@/schematic/node/pumps/RollerVane";
import { SYMBOL as SCREW } from "@/schematic/node/pumps/Screw";
import { SYMBOL as TURBO } from "@/schematic/node/pumps/Turbo";
import { SYMBOL as VACUUM } from "@/schematic/node/pumps/Vacuum";

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

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  cavity_pump: { name: "Cavity", label: "Cavity pump" },
  centrifugal_compressor: {
    name: "Centrifugal compressor",
    label: "Centrifugal compressor",
  },
  compressor: { name: "Compressor", label: "Compressor" },
  diaphragm_pump: { name: "Diaphragm pump", label: "Diaphragm pump" },
  ejection_pump: { name: "Ejection", label: "Ejection pump" },
  ejector_compressor: { name: "Ejector compressor", label: "Ejector compressor" },
  liquid_ring_compressor: {
    name: "Liquid ring compressor",
    label: "Liquid ring compressor",
  },
  piston_pump: { name: "Piston", label: "Piston pump" },
  pump: { name: "Pump", label: "Pump" },
  roller_vane_compressor: {
    name: "Roller vane compressor",
    label: "Roller vane compressor",
  },
  screw_pump: { name: "Screw", label: "Screw pump" },
  turbo_compressor: { name: "Turbo compressor", label: "Turbo compressor" },
  vacuum_pump: { name: "Vacuum", label: "Vacuum pump" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
