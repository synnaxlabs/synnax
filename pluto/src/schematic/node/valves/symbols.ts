// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import {
  createDummyToggle,
  createStatic,
  createToggle,
  withoutToggleConfig,
} from "@/schematic/node/common/create";
import { Toggle } from "@/schematic/node/common/toggle";
import type { Spec } from "@/schematic/node/spec";
import { Angled, SYMBOL as ANGLED } from "@/schematic/node/valves/Angled";
import {
  AngledRelief,
  SYMBOL as ANGLED_RELIEF,
} from "@/schematic/node/valves/AngledRelief";
import {
  AngledSpringLoadedRelief,
  SYMBOL as ANGLED_SPRING_LOADED_RELIEF,
} from "@/schematic/node/valves/AngledSpringLoadedRelief";
import { Ball, SYMBOL as BALL } from "@/schematic/node/valves/Ball";
import { Breather, SYMBOL as BREATHER } from "@/schematic/node/valves/Breather";
import {
  ButterflyOne,
  SYMBOL as BUTTERFLY_ONE,
} from "@/schematic/node/valves/ButterflyOne";
import {
  ButterflyTwo,
  SYMBOL as BUTTERFLY_TWO,
} from "@/schematic/node/valves/ButterflyTwo";
import { Check, SYMBOL as CHECK } from "@/schematic/node/valves/Check";
import {
  CheckWithArrow,
  SYMBOL as CHECK_WITH_ARROW,
} from "@/schematic/node/valves/CheckWithArrow";
import {
  ElectricRegulator,
  SYMBOL as ELECTRIC_REGULATOR,
} from "@/schematic/node/valves/ElectricRegulator";
import {
  ElectricRegulatorMotorized,
  SYMBOL as ELECTRIC_REGULATOR_MOTORIZED,
} from "@/schematic/node/valves/ElectricRegulatorMotorized";
import { FourWay, SYMBOL as FOUR_WAY } from "@/schematic/node/valves/FourWay";
import { Gate, SYMBOL as GATE } from "@/schematic/node/valves/Gate";
import { IsoCheck, SYMBOL as ISO_CHECK } from "@/schematic/node/valves/IsoCheck";
import { Manual, SYMBOL as MANUAL } from "@/schematic/node/valves/Manual";
import { Needle, SYMBOL as NEEDLE } from "@/schematic/node/valves/Needle";
import { Regulator, SYMBOL as REGULATOR } from "@/schematic/node/valves/Regulator";
import {
  RegulatorManual,
  SYMBOL as REGULATOR_MANUAL,
} from "@/schematic/node/valves/RegulatorManual";
import { Relief, SYMBOL as RELIEF } from "@/schematic/node/valves/Relief";
import {
  NORMALLY_OPEN_SYMBOL,
  Solenoid,
  SYMBOL as SOLENOID,
} from "@/schematic/node/valves/Solenoid";
import { SolenoidForm } from "@/schematic/node/valves/SolenoidForm";
import {
  SpringLoadedRelief,
  SYMBOL as SPRING_LOADED_RELIEF,
} from "@/schematic/node/valves/SpringLoadedRelief";
import { SYMBOL as THREE_WAY, ThreeWay } from "@/schematic/node/valves/ThreeWay";
import {
  SYMBOL as THREE_WAY_BALL,
  ThreeWayBall,
} from "@/schematic/node/valves/ThreeWayBall";
import { SYMBOL as VALVE, Valve } from "@/schematic/node/valves/Valve";

const angled = createToggle({
  variant: "angled_valve",
  name: "Angled",
  label: "Angled valve",
  Primitive: Angled,
});
const angledRelief = createDummyToggle({
  variant: "angled_relief_valve",
  name: "Angled relief",
  label: "Angled relief valve",
  Primitive: AngledRelief,
});
const angledSpringLoadedRelief = createDummyToggle({
  variant: "angled_spring_loaded_relief_valve",
  name: "Angled spring loaded relief",
  label: "Angled spring loaded relief valve",
  Primitive: AngledSpringLoadedRelief,
});
const ball = createToggle({
  variant: "ball_valve",
  name: "Ball",
  label: "Ball valve",
  Primitive: Ball,
});
const breather = createDummyToggle({
  variant: "breather_valve",
  name: "Breather",
  label: "Breather valve",
  Primitive: Breather,
});
const butterflyOne = createToggle({
  variant: "butterfly_valve_one",
  name: "Butterfly (remote)",
  label: "Butterfly valve (remote)",
  Primitive: ButterflyOne,
});
const butterflyTwo = createToggle({
  variant: "butterfly_valve_two",
  name: "Butterfly (manual)",
  label: "Butterfly valve (manual)",
  Primitive: ButterflyTwo,
});
const check = createStatic({
  variant: "check_valve",
  name: "Check",
  label: "Check valve",
  Primitive: Check,
});
const checkWithArrow = createStatic({
  variant: "check_valve_with_arrow",
  name: "Check (arrow)",
  label: "Check valve",
  Primitive: CheckWithArrow,
});
const electricRegulator = createStatic({
  variant: "electric_regulator",
  name: "Electric",
  label: "Electric regulator",
  Primitive: ElectricRegulator,
});
const electricRegulatorMotorized = createStatic({
  variant: "electric_regulator_motorized",
  name: "Motorized",
  label: "Electric regulator motorized",
  Primitive: ElectricRegulatorMotorized,
});
const fourWay = createToggle({
  variant: "four_way_valve",
  name: "Four way",
  label: "Four way valve",
  Primitive: FourWay,
});
const gate = createToggle({
  variant: "gate_valve",
  name: "Gate",
  label: "Gate valve",
  Primitive: Gate,
});
const isoCheck = createStatic({
  variant: "iso_check_valve",
  name: "ISO check",
  label: "ISO check valve",
  Primitive: IsoCheck,
});
const manual = createDummyToggle({
  variant: "manual_valve",
  name: "Manual",
  label: "Manual valve",
  Primitive: Manual,
});
const needle = createDummyToggle({
  variant: "needle_valve",
  name: "Needle",
  label: "Needle valve",
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
  label: "Manual regulator",
  Primitive: RegulatorManual,
});
const relief = createDummyToggle({
  variant: "relief_valve",
  name: "Relief",
  label: "Relief valve",
  Primitive: Relief,
});
type SolenoidConfig = schematic.SolenoidValveNodeConfig;
// Solenoid is the one valve whose config carries an extra `normallyOpen` field, so it
// is built directly rather than through defineToggle.
const solenoidSpec: Spec<"solenoid_valve", SolenoidConfig> = {
  key: "solenoid_valve",
  name: "Solenoid",
  Form: SolenoidForm,
  Node: Toggle.createToggle<SolenoidConfig>(Solenoid),
  Preview: withoutToggleConfig(Solenoid),
  zIndex: 4,
};
const springLoadedRelief = createDummyToggle({
  variant: "spring_loaded_relief_valve",
  name: "Spring loaded relief",
  label: "Spring loaded relief valve",
  Primitive: SpringLoadedRelief,
});
const threeWay = createToggle({
  variant: "three_way_valve",
  name: "Three way",
  Primitive: ThreeWay,
});
const threeWayBall = createToggle({
  variant: "three_way_ball_valve",
  name: "Three-way ball",
  label: "Three-way ball valve",
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

/** BUILT_INS maps the name of each built-in symbol to its spec. */
export const BUILT_INS = {
  angled_relief_valve: ANGLED_RELIEF,
  angled_spring_loaded_relief_valve: ANGLED_SPRING_LOADED_RELIEF,
  angled_valve: ANGLED,
  ball_valve: BALL,
  breather_valve: BREATHER,
  butterfly_valve_one: BUTTERFLY_ONE,
  butterfly_valve_two: BUTTERFLY_TWO,
  check_valve: CHECK,
  check_valve_with_arrow: CHECK_WITH_ARROW,
  electric_regulator: ELECTRIC_REGULATOR,
  electric_regulator_motorized: ELECTRIC_REGULATOR_MOTORIZED,
  four_way_valve: FOUR_WAY,
  gate_valve: GATE,
  iso_check_valve: ISO_CHECK,
  manual_valve: MANUAL,
  needle_valve: NEEDLE,
  normally_open_solenoid_valve: NORMALLY_OPEN_SYMBOL,
  regulator: REGULATOR,
  regulator_manual: REGULATOR_MANUAL,
  relief_valve: RELIEF,
  solenoid_valve: SOLENOID,
  spring_loaded_relief_valve: SPRING_LOADED_RELIEF,
  three_way_ball_valve: THREE_WAY_BALL,
  three_way_valve: THREE_WAY,
  valve: VALVE,
} as const satisfies Record<string, schematic.symbol.Spec>;
