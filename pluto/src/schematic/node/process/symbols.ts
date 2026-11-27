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
import { SYMBOL as AGITATOR } from "@/schematic/node/process/Agitator";
import { SYMBOL as CROSS_BEAM_AGITATOR } from "@/schematic/node/process/CrossBeamAgitator";
import { SYMBOL as FLAT_BLADE_AGITATOR } from "@/schematic/node/process/FlatBladeAgitator";
import { SYMBOL as HEAT_EXCHANGER_GENERAL } from "@/schematic/node/process/HeatExchangerGeneral";
import { SYMBOL as HEAT_EXCHANGER_M } from "@/schematic/node/process/HeatExchangerM";
import { SYMBOL as HEAT_EXCHANGER_STRAIGHT_TUBE } from "@/schematic/node/process/HeatExchangerStraightTube";
import { SYMBOL as HELICAL_AGITATOR } from "@/schematic/node/process/HelicalAgitator";
import { SYMBOL as PADDLE_AGITATOR } from "@/schematic/node/process/PaddleAgitator";
import { SYMBOL as PROPELLER_AGITATOR } from "@/schematic/node/process/PropellerAgitator";
import { SYMBOL as ROTARY_MIXER } from "@/schematic/node/process/RotaryMixer";
import { SYMBOL as STATIC_MIXER } from "@/schematic/node/process/StaticMixer";

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  agitator: AGITATOR,
  cross_beam_agitator: CROSS_BEAM_AGITATOR,
  flat_blade_agitator: FLAT_BLADE_AGITATOR,
  heat_exchanger_general: HEAT_EXCHANGER_GENERAL,
  heat_exchanger_m: HEAT_EXCHANGER_M,
  heat_exchanger_straight_tube: HEAT_EXCHANGER_STRAIGHT_TUBE,
  helical_agitator: HELICAL_AGITATOR,
  paddle_agitator: PADDLE_AGITATOR,
  propeller_agitator: PROPELLER_AGITATOR,
  rotary_mixer: ROTARY_MIXER,
  static_mixer: STATIC_MIXER,
} as const satisfies Record<string, schematic.symbol.Spec>;

/** NAMES maps the name of each built-in symbol to its names. */
export const NAMES = {
  agitator: { name: "Agitator", label: "Agitator" },
  cross_beam_agitator: { name: "Cross beam agitator", label: "Cross beam agitator" },
  flat_blade_agitator: { name: "Flat blade agitator", label: "Flat blade agitator" },
  heat_exchanger_general: { name: "Heat exchanger", label: "General heat exchanger" },
  heat_exchanger_m: { name: "M-type heat exchanger", label: "M heat exchanger" },
  heat_exchanger_straight_tube: {
    name: "Straight tube heat exchanger",
    label: "Straight tube heat exchanger",
  },
  helical_agitator: { name: "Helical agitator", label: "Helical agitator" },
  paddle_agitator: { name: "Paddle agitator", label: "Paddle agitator" },
  propeller_agitator: { name: "Propeller agitator", label: "Propeller agitator" },
  rotary_mixer: { name: "Rotary mixer", label: "Rotary mixer" },
  static_mixer: { name: "Static mixer", label: "Static mixer" },
} as const satisfies Record<keyof typeof BUILT_INS, Names>;
