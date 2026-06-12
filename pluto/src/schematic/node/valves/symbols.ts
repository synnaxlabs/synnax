// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";

import {
  createDummyToggle,
  createStatic,
  createToggle,
} from "@/schematic/node/common/create";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import { type Spec } from "@/schematic/node/spec";
import { Angled } from "@/schematic/node/valves/Angled";
import { AngledRelief } from "@/schematic/node/valves/AngledRelief";
import { AngledSpringLoadedRelief } from "@/schematic/node/valves/AngledSpringLoadedRelief";
import { Ball } from "@/schematic/node/valves/Ball";
import { Breather } from "@/schematic/node/valves/Breather";
import { ButterflyOne } from "@/schematic/node/valves/ButterflyOne";
import { ButterflyTwo } from "@/schematic/node/valves/ButterflyTwo";
import { Check } from "@/schematic/node/valves/Check";
import { CheckWithArrow } from "@/schematic/node/valves/CheckWithArrow";
import { ElectricRegulator } from "@/schematic/node/valves/ElectricRegulator";
import { ElectricRegulatorMotorized } from "@/schematic/node/valves/ElectricRegulatorMotorized";
import { FourWay } from "@/schematic/node/valves/FourWay";
import { Gate } from "@/schematic/node/valves/Gate";
import { IsoCheck } from "@/schematic/node/valves/IsoCheck";
import { Manual } from "@/schematic/node/valves/Manual";
import { Needle } from "@/schematic/node/valves/Needle";
import { Regulator } from "@/schematic/node/valves/Regulator";
import { RegulatorManual } from "@/schematic/node/valves/RegulatorManual";
import { Relief } from "@/schematic/node/valves/Relief";
import { Solenoid } from "@/schematic/node/valves/Solenoid";
import { SpringLoadedRelief } from "@/schematic/node/valves/SpringLoadedRelief";
import { ThreeWay } from "@/schematic/node/valves/ThreeWay";
import { ThreeWayBall } from "@/schematic/node/valves/ThreeWayBall";
import { Valve } from "@/schematic/node/valves/Valve";

const angled = createToggle({
  variant: "angled_valve",
  name: "Angled",
  label: "Angled Valve",
  Primitive: Angled,
});
const angledRelief = createDummyToggle({
  variant: "angled_relief_valve",
  name: "Angled Relief",
  label: "Angled Relief Valve",
  Primitive: AngledRelief,
});
const angledSpringLoadedRelief = createDummyToggle({
  variant: "angled_spring_loaded_relief_valve",
  name: "Angled Spring Loaded Relief",
  label: "Angled Spring Loaded Relief Valve",
  Primitive: AngledSpringLoadedRelief,
});
const ball = createToggle({
  variant: "ball_valve",
  name: "Ball",
  label: "Ball Valve",
  Primitive: Ball,
});
const breather = createDummyToggle({
  variant: "breather_valve",
  name: "Breather",
  label: "Breather Valve",
  Primitive: Breather,
});
const butterflyOne = createToggle({
  variant: "butterfly_valve_one",
  name: "Butterfly (Remote)",
  label: "Butterfly Valve (Remote)",
  Primitive: ButterflyOne,
});
const butterflyTwo = createToggle({
  variant: "butterfly_valve_two",
  name: "Butterfly (Manual)",
  label: "Butterfly Valve (Manual)",
  Primitive: ButterflyTwo,
});
const check = createStatic({
  variant: "check_valve",
  name: "Check",
  label: "Check Valve",
  Primitive: Check,
});
const checkWithArrow = createStatic({
  variant: "check_valve_with_arrow",
  name: "Check (Arrow)",
  label: "Check Valve",
  Primitive: CheckWithArrow,
});
const electricRegulator = createStatic({
  variant: "electric_regulator",
  name: "Electric",
  label: "Electric Regulator",
  Primitive: ElectricRegulator,
});
const electricRegulatorMotorized = createStatic({
  variant: "electric_regulator_motorized",
  name: "Motorized",
  label: "Electric Regulator Motorized",
  Primitive: ElectricRegulatorMotorized,
});
const fourWay = createToggle({
  variant: "four_way_valve",
  name: "Four Way",
  label: "Four Way Valve",
  Primitive: FourWay,
});
const gate = createToggle({
  variant: "gate_valve",
  name: "Gate",
  label: "Gate Valve",
  Primitive: Gate,
});
const isoCheck = createStatic({
  variant: "iso_check_valve",
  name: "ISO Check",
  label: "ISO Check Valve",
  Primitive: IsoCheck,
});
const manual = createDummyToggle({
  variant: "manual_valve",
  name: "Manual",
  label: "Manual Valve",
  Primitive: Manual,
});
const needle = createDummyToggle({
  variant: "needle_valve",
  name: "Needle",
  label: "Needle Valve",
  Primitive: Needle,
});
const regulator = createStatic({
  variant: "regulator",
  name: "Regulator",
  Primitive: Regulator,
});
const regulatorManual = createStatic({
  variant: "regulator_manual",
  name: "Manual",
  label: "Manual Regulator",
  Primitive: RegulatorManual,
});
const relief = createDummyToggle({
  variant: "relief_valve",
  name: "Relief",
  label: "Relief Valve",
  Primitive: Relief,
});
type SolenoidConfig = schematic.NodeConfigSolenoidValve;
// Solenoid is the one valve whose config carries an extra `normallyOpen` field, so it
// is built directly rather than through defineToggle. The spec is given an explicit
// Spec annotation (not `satisfies`) to widen Form to FC<FormProps> — otherwise the
// concrete ToggleForm type leaks ToggleFormProps into REGISTRY.
const solenoidSpec: Spec<"solenoid_valve", SolenoidConfig> = {
  key: "solenoid_valve",
  name: "Solenoid",
  Form: Form.ToggleForm,
  Node: Toggle.createToggle<SolenoidConfig>(Solenoid),
  Preview: Solenoid,
  defaultConfig: (): SolenoidConfig => ({
    variant: "solenoid_valve",
    color: color.ZERO,
    label: Label.defaultConfig("Solenoid Valve"),
    normallyOpen: false,
    ...Primitive.ZERO_PROPS,
    ...Toggle.ZERO_TOGGLE_DEFAULTS,
  }),
  zIndex: 4,
};
const springLoadedRelief = createDummyToggle({
  variant: "spring_loaded_relief_valve",
  name: "Spring Loaded Relief",
  label: "Spring Loaded Relief Valve",
  Primitive: SpringLoadedRelief,
});
const threeWay = createToggle({
  variant: "three_way_valve",
  name: "Three Way",
  Primitive: ThreeWay,
});
const threeWayBall = createToggle({
  variant: "three_way_ball_valve",
  name: "Three-Way Ball",
  label: "Three-Way Ball Valve",
  Primitive: ThreeWayBall,
});
const valve = createToggle({
  variant: "valve",
  name: "Generic",
  label: "Valve",
  Primitive: Valve,
});

export const REGISTRY = {
  angled_valve: angled.spec,
  angled_relief_valve: angledRelief.spec,
  angled_spring_loaded_relief_valve: angledSpringLoadedRelief.spec,
  ball_valve: ball.spec,
  breather_valve: breather.spec,
  butterfly_valve_one: butterflyOne.spec,
  butterfly_valve_two: butterflyTwo.spec,
  check_valve: check.spec,
  check_valve_with_arrow: checkWithArrow.spec,
  electric_regulator: electricRegulator.spec,
  electric_regulator_motorized: electricRegulatorMotorized.spec,
  four_way_valve: fourWay.spec,
  gate_valve: gate.spec,
  iso_check_valve: isoCheck.spec,
  manual_valve: manual.spec,
  needle_valve: needle.spec,
  regulator: regulator.spec,
  regulator_manual: regulatorManual.spec,
  relief_valve: relief.spec,
  solenoid_valve: solenoidSpec,
  spring_loaded_relief_valve: springLoadedRelief.spec,
  three_way_valve: threeWay.spec,
  three_way_ball_valve: threeWayBall.spec,
  valve: valve.spec,
} as const;
