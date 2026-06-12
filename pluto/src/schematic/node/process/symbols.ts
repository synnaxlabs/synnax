// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createStatic, createToggle } from "@/schematic/node/common/create";
import { Agitator } from "@/schematic/node/process/Agitator";
import { CrossBeamAgitator } from "@/schematic/node/process/CrossBeamAgitator";
import { FlatBladeAgitator } from "@/schematic/node/process/FlatBladeAgitator";
import { HeatExchangerGeneral } from "@/schematic/node/process/HeatExchangerGeneral";
import { HeatExchangerM } from "@/schematic/node/process/HeatExchangerM";
import { HeatExchangerStraightTube } from "@/schematic/node/process/HeatExchangerStraightTube";
import { HelicalAgitator } from "@/schematic/node/process/HelicalAgitator";
import { PaddleAgitator } from "@/schematic/node/process/PaddleAgitator";
import { PropellerAgitator } from "@/schematic/node/process/PropellerAgitator";
import { RotaryMixer } from "@/schematic/node/process/RotaryMixer";
import { StaticMixer } from "@/schematic/node/process/StaticMixer";

const agitator = createToggle({
  variant: "agitator",
  name: "Agitator",
  Primitive: Agitator,
});
const crossBeamAgitator = createToggle({
  variant: "cross_beam_agitator",
  name: "Cross Beam Agitator",
  Primitive: CrossBeamAgitator,
});
const flatBladeAgitator = createToggle({
  variant: "flat_blade_agitator",
  name: "Flat Blade Agitator",
  Primitive: FlatBladeAgitator,
});
const heatExchangerGeneral = createStatic({
  variant: "heat_exchanger_general",
  name: "Heat Exchanger",
  label: "General Heat Exchanger",
  Primitive: HeatExchangerGeneral,
});
const heatExchangerM = createStatic({
  variant: "heat_exchanger_m",
  name: "M-Type Heat Exchanger",
  label: "M Heat Exchanger",
  Primitive: HeatExchangerM,
});
const heatExchangerStraightTube = createStatic({
  variant: "heat_exchanger_straight_tube",
  name: "Straight Tube Heat Exchanger",
  Primitive: HeatExchangerStraightTube,
});
const helicalAgitator = createToggle({
  variant: "helical_agitator",
  name: "Helical Agitator",
  Primitive: HelicalAgitator,
});
const paddleAgitator = createToggle({
  variant: "paddle_agitator",
  name: "Paddle Agitator",
  Primitive: PaddleAgitator,
});
const propellerAgitator = createToggle({
  variant: "propeller_agitator",
  name: "Propeller Agitator",
  Primitive: PropellerAgitator,
});
const rotaryMixer = createToggle({
  variant: "rotary_mixer",
  name: "Rotary Mixer",
  Primitive: RotaryMixer,
});
const staticMixer = createStatic({
  variant: "static_mixer",
  name: "Static Mixer",
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
