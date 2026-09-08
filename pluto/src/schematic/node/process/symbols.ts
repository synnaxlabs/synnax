// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

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
  variant: "crossBeamAgitator",
  name: "Cross beam agitator",
  Primitive: CrossBeamAgitator,
});
const flatBladeAgitator = createToggle({
  variant: "flatBladeAgitator",
  name: "Flat blade agitator",
  Primitive: FlatBladeAgitator,
});
const heatExchangerGeneral = createStatic({
  variant: "heatExchangerGeneral",
  name: "Heat exchanger",
  label: "General heat exchanger",
  Primitive: HeatExchangerGeneral,
});
const heatExchangerM = createStatic({
  variant: "heatExchangerM",
  name: "M-type heat exchanger",
  label: "M heat exchanger",
  Primitive: HeatExchangerM,
});
const heatExchangerStraightTube = createStatic({
  variant: "heatExchangerStraightTube",
  name: "Straight tube heat exchanger",
  Primitive: HeatExchangerStraightTube,
});
const helicalAgitator = createToggle({
  variant: "helicalAgitator",
  name: "Helical agitator",
  Primitive: HelicalAgitator,
});
const paddleAgitator = createToggle({
  variant: "paddleAgitator",
  name: "Paddle agitator",
  Primitive: PaddleAgitator,
});
const propellerAgitator = createToggle({
  variant: "propellerAgitator",
  name: "Propeller agitator",
  Primitive: PropellerAgitator,
});
const rotaryMixer = createToggle({
  variant: "rotaryMixer",
  name: "Rotary mixer",
  Primitive: RotaryMixer,
});
const staticMixer = createStatic({
  variant: "staticMixer",
  name: "Static mixer",
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
