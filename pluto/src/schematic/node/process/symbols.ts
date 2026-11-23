// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { createStatic, createToggle } from "@/schematic/node/common/create";
import { Agitator, SYMBOL as AGITATOR } from "@/schematic/node/process/Agitator";
import {
  CrossBeamAgitator,
  SYMBOL as CROSS_BEAM_AGITATOR,
} from "@/schematic/node/process/CrossBeamAgitator";
import {
  FlatBladeAgitator,
  SYMBOL as FLAT_BLADE_AGITATOR,
} from "@/schematic/node/process/FlatBladeAgitator";
import {
  HeatExchangerGeneral,
  SYMBOL as HEAT_EXCHANGER_GENERAL,
} from "@/schematic/node/process/HeatExchangerGeneral";
import {
  HeatExchangerM,
  SYMBOL as HEAT_EXCHANGER_M,
} from "@/schematic/node/process/HeatExchangerM";
import {
  HeatExchangerStraightTube,
  SYMBOL as HEAT_EXCHANGER_STRAIGHT_TUBE,
} from "@/schematic/node/process/HeatExchangerStraightTube";
import {
  HelicalAgitator,
  SYMBOL as HELICAL_AGITATOR,
} from "@/schematic/node/process/HelicalAgitator";
import {
  PaddleAgitator,
  SYMBOL as PADDLE_AGITATOR,
} from "@/schematic/node/process/PaddleAgitator";
import {
  PropellerAgitator,
  SYMBOL as PROPELLER_AGITATOR,
} from "@/schematic/node/process/PropellerAgitator";
import {
  RotaryMixer,
  SYMBOL as ROTARY_MIXER,
} from "@/schematic/node/process/RotaryMixer";
import {
  StaticMixer,
  SYMBOL as STATIC_MIXER,
} from "@/schematic/node/process/StaticMixer";

const agitator = createToggle({
  variant: "agitator",
  name: "Agitator",
  Primitive: Agitator,
});
const crossBeamAgitator = createToggle({
  variant: "cross_beam_agitator",
  name: "Cross beam agitator",
  Primitive: CrossBeamAgitator,
});
const flatBladeAgitator = createToggle({
  variant: "flat_blade_agitator",
  name: "Flat blade agitator",
  Primitive: FlatBladeAgitator,
});
const heatExchangerGeneral = createStatic({
  variant: "heat_exchanger_general",
  name: "Heat exchanger",
  label: "General heat exchanger",
  Primitive: HeatExchangerGeneral,
});
const heatExchangerM = createStatic({
  variant: "heat_exchanger_m",
  name: "M-type heat exchanger",
  label: "M heat exchanger",
  Primitive: HeatExchangerM,
});
const heatExchangerStraightTube = createStatic({
  variant: "heat_exchanger_straight_tube",
  name: "Straight tube heat exchanger",
  Primitive: HeatExchangerStraightTube,
});
const helicalAgitator = createToggle({
  variant: "helical_agitator",
  name: "Helical agitator",
  Primitive: HelicalAgitator,
});
const paddleAgitator = createToggle({
  variant: "paddle_agitator",
  name: "Paddle agitator",
  Primitive: PaddleAgitator,
});
const propellerAgitator = createToggle({
  variant: "propeller_agitator",
  name: "Propeller agitator",
  Primitive: PropellerAgitator,
});
const rotaryMixer = createToggle({
  variant: "rotary_mixer",
  name: "Rotary mixer",
  Primitive: RotaryMixer,
});
const staticMixer = createStatic({
  variant: "static_mixer",
  name: "Static mixer",
  Primitive: StaticMixer,
});

export const REGISTRY = {
  agitator: agitator.spec,
  cross_beam_agitator: crossBeamAgitator.spec,
  flat_blade_agitator: flatBladeAgitator.spec,
  heat_exchanger_general: heatExchangerGeneral.spec,
  heat_exchanger_m: heatExchangerM.spec,
  heat_exchanger_straight_tube: heatExchangerStraightTube.spec,
  helical_agitator: helicalAgitator.spec,
  paddle_agitator: paddleAgitator.spec,
  propeller_agitator: propellerAgitator.spec,
  rotary_mixer: rotaryMixer.spec,
  static_mixer: staticMixer.spec,
} as const;

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
