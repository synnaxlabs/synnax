// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { defineStatic, defineToggle } from "@/schematic/node/define";
import { Primitive as Agitator } from "@/schematic/node/process/Agitator";
import { Primitive as CrossBeamAgitator } from "@/schematic/node/process/CrossBeamAgitator";
import { Primitive as FlatBladeAgitator } from "@/schematic/node/process/FlatBladeAgitator";
import { Primitive as HeatExchangerGeneral } from "@/schematic/node/process/HeatExchangerGeneral";
import { Primitive as HeatExchangerM } from "@/schematic/node/process/HeatExchangerM";
import { Primitive as HeatExchangerStraightTube } from "@/schematic/node/process/HeatExchangerStraightTube";
import { Primitive as HelicalAgitator } from "@/schematic/node/process/HelicalAgitator";
import { Primitive as PaddleAgitator } from "@/schematic/node/process/PaddleAgitator";
import { Primitive as PropellerAgitator } from "@/schematic/node/process/PropellerAgitator";
import { Primitive as RotaryMixer } from "@/schematic/node/process/RotaryMixer";
import { Primitive as StaticMixer } from "@/schematic/node/process/StaticMixer";

const agitator = defineToggle({
  variant: "agitator",
  name: "Agitator",
  label: "Agitator",
  Primitive: Agitator,
});
const crossBeamAgitator = defineToggle({
  variant: "crossBeamAgitator",
  name: "Cross Beam Agitator",
  label: "Cross Beam Agitator",
  Primitive: CrossBeamAgitator,
});
const flatBladeAgitator = defineToggle({
  variant: "flatBladeAgitator",
  name: "Flat Blade Agitator",
  label: "Flat Blade Agitator",
  Primitive: FlatBladeAgitator,
});
const heatExchangerGeneral = defineStatic({
  variant: "heatExchangerGeneral",
  name: "Heat Exchanger",
  label: "General Heat Exchanger",
  Primitive: HeatExchangerGeneral,
});
const heatExchangerM = defineStatic({
  variant: "heatExchangerM",
  name: "M-Type Heat Exchanger",
  label: "M Heat Exchanger",
  Primitive: HeatExchangerM,
});
const heatExchangerStraightTube = defineStatic({
  variant: "heatExchangerStraightTube",
  name: "Straight Tube Heat Exchanger",
  label: "Straight Tube Heat Exchanger",
  Primitive: HeatExchangerStraightTube,
});
const helicalAgitator = defineToggle({
  variant: "helicalAgitator",
  name: "Helical Agitator",
  label: "Helical Agitator",
  Primitive: HelicalAgitator,
});
const paddleAgitator = defineToggle({
  variant: "paddleAgitator",
  name: "Paddle Agitator",
  label: "Paddle Agitator",
  Primitive: PaddleAgitator,
});
const propellerAgitator = defineToggle({
  variant: "propellerAgitator",
  name: "Propeller Agitator",
  label: "Propeller Agitator",
  Primitive: PropellerAgitator,
});
const rotaryMixer = defineToggle({
  variant: "rotaryMixer",
  name: "Rotary Mixer",
  label: "Rotary Mixer",
  Primitive: RotaryMixer,
});
const staticMixer = defineStatic({
  variant: "staticMixer",
  name: "Static Mixer",
  label: "Static Mixer",
  Primitive: StaticMixer,
});

export const REGISTRY = {
  agitator: agitator.spec,
  crossBeamAgitator: crossBeamAgitator.spec,
  flatBladeAgitator: flatBladeAgitator.spec,
  heatExchangerGeneral: heatExchangerGeneral.spec,
  heatExchangerM: heatExchangerM.spec,
  heatExchangerStraightTube: heatExchangerStraightTube.spec,
  helicalAgitator: helicalAgitator.spec,
  paddleAgitator: paddleAgitator.spec,
  propellerAgitator: propellerAgitator.spec,
  rotaryMixer: rotaryMixer.spec,
  staticMixer: staticMixer.spec,
} as const;

export const configZ = z.discriminatedUnion("variant", [
  agitator.configZ,
  crossBeamAgitator.configZ,
  flatBladeAgitator.configZ,
  heatExchangerGeneral.configZ,
  heatExchangerM.configZ,
  heatExchangerStraightTube.configZ,
  helicalAgitator.configZ,
  paddleAgitator.configZ,
  propellerAgitator.configZ,
  rotaryMixer.configZ,
  staticMixer.configZ,
]);
