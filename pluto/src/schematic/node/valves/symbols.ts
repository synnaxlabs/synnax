// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { z } from "zod";

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
import { type Theming } from "@/theming";

const angled = createToggle({
  variant: "angledValve",
  name: "Angled",
  label: "Angled Valve",
  Primitive: Angled,
});
const angledRelief = createDummyToggle({
  variant: "angledReliefValve",
  name: "Angled Relief",
  label: "Angled Relief Valve",
  Primitive: AngledRelief,
});
const angledSpringLoadedRelief = createDummyToggle({
  variant: "angledSpringLoadedReliefValve",
  name: "Angled Spring Loaded Relief",
  label: "Angled Spring Loaded Relief Valve",
  Primitive: AngledSpringLoadedRelief,
});
const ball = createToggle({
  variant: "ballValve",
  name: "Ball",
  label: "Ball Valve",
  Primitive: Ball,
});
const breather = createDummyToggle({
  variant: "breatherValve",
  name: "Breather",
  label: "Breather Valve",
  Primitive: Breather,
});
const butterflyOne = createToggle({
  variant: "butterflyValveOne",
  name: "Butterfly (Remote)",
  label: "Butterfly Valve (Remote)",
  Primitive: ButterflyOne,
});
const butterflyTwo = createToggle({
  variant: "butterflyValveTwo",
  name: "Butterfly (Manual)",
  label: "Butterfly Valve (Manual)",
  Primitive: ButterflyTwo,
});
const check = createStatic({
  variant: "checkValve",
  name: "Check",
  label: "Check Valve",
  Primitive: Check,
});
const checkWithArrow = createStatic({
  variant: "checkValveWithArrow",
  name: "Check (Arrow)",
  label: "Check Valve",
  Primitive: CheckWithArrow,
});
const electricRegulator = createStatic({
  variant: "electricRegulator",
  name: "Electric",
  label: "Electric Regulator",
  Primitive: ElectricRegulator,
});
const electricRegulatorMotorized = createStatic({
  variant: "electricRegulatorMotorized",
  name: "Motorized",
  label: "Electric Regulator Motorized",
  Primitive: ElectricRegulatorMotorized,
});
const fourWay = createToggle({
  variant: "fourWayValve",
  name: "Four Way",
  label: "Four Way Valve",
  Primitive: FourWay,
});
const gate = createToggle({
  variant: "gateValve",
  name: "Gate",
  label: "Gate Valve",
  Primitive: Gate,
});
const isoCheck = createStatic({
  variant: "isoCheckValve",
  name: "ISO Check",
  label: "ISO Check Valve",
  Primitive: IsoCheck,
});
const manual = createDummyToggle({
  variant: "manualValve",
  name: "Manual",
  label: "Manual Valve",
  Primitive: Manual,
});
const needle = createDummyToggle({
  variant: "needleValve",
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
  variant: "regulatorManual",
  name: "Manual",
  label: "Manual Regulator",
  Primitive: RegulatorManual,
});
const relief = createDummyToggle({
  variant: "reliefValve",
  name: "Relief",
  label: "Relief Valve",
  Primitive: Relief,
});
const solenoidConfigZ = Toggle.toggleConfigZ.extend({
  variant: z.literal("solenoidValve"),
  color: color.crudeZ.optional(),
  scale: z.number().optional(),
  normallyOpen: z.boolean().optional(),
});
type SolenoidConfig = z.infer<typeof solenoidConfigZ>;
// Solenoid is the one valve whose config carries an extra `normallyOpen` field, so it
// is built directly rather than through defineToggle. The spec is given an explicit
// Spec annotation (not `satisfies`) to widen Form to FC<FormProps> — otherwise the
// concrete ToggleForm type leaks ToggleFormProps into REGISTRY.
const solenoidSpec: Spec<"solenoidValve", SolenoidConfig> = {
  key: "solenoidValve",
  name: "Solenoid",
  Form: Form.ToggleForm,
  Node: Toggle.createToggle<SolenoidConfig>(Solenoid),
  Preview: Solenoid,
  defaultConfig: (_t: Theming.Theme): SolenoidConfig => ({
    variant: "solenoidValve",
    color: color.ZERO,
    label: Label.defaultConfig("Solenoid Valve"),
    normallyOpen: false,
    ...Primitive.ZERO_PROPS,
    ...Toggle.ZERO_TOGGLE_DEFAULTS,
  }),
  zIndex: 4,
};
const springLoadedRelief = createDummyToggle({
  variant: "springLoadedReliefValve",
  name: "Spring Loaded Relief",
  label: "Spring Loaded Relief Valve",
  Primitive: SpringLoadedRelief,
});
const threeWay = createToggle({
  variant: "threeWayValve",
  name: "Three Way",
  Primitive: ThreeWay,
});
const threeWayBall = createToggle({
  variant: "threeWayBallValve",
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
  angledValve: angled.spec,
  angledReliefValve: angledRelief.spec,
  angledSpringLoadedReliefValve: angledSpringLoadedRelief.spec,
  ballValve: ball.spec,
  breatherValve: breather.spec,
  butterflyValveOne: butterflyOne.spec,
  butterflyValveTwo: butterflyTwo.spec,
  checkValve: check.spec,
  checkValveWithArrow: checkWithArrow.spec,
  electricRegulator: electricRegulator.spec,
  electricRegulatorMotorized: electricRegulatorMotorized.spec,
  fourWayValve: fourWay.spec,
  gateValve: gate.spec,
  isoCheckValve: isoCheck.spec,
  manualValve: manual.spec,
  needleValve: needle.spec,
  regulator: regulator.spec,
  regulatorManual: regulatorManual.spec,
  reliefValve: relief.spec,
  solenoidValve: solenoidSpec,
  springLoadedReliefValve: springLoadedRelief.spec,
  threeWayValve: threeWay.spec,
  threeWayBallValve: threeWayBall.spec,
  valve: valve.spec,
} as const;

export const configZ = z.discriminatedUnion("variant", [
  angled.configZ,
  angledRelief.configZ,
  angledSpringLoadedRelief.configZ,
  ball.configZ,
  breather.configZ,
  butterflyOne.configZ,
  butterflyTwo.configZ,
  check.configZ,
  checkWithArrow.configZ,
  electricRegulator.configZ,
  electricRegulatorMotorized.configZ,
  fourWay.configZ,
  gate.configZ,
  isoCheck.configZ,
  manual.configZ,
  needle.configZ,
  regulator.configZ,
  regulatorManual.configZ,
  relief.configZ,
  solenoidConfigZ,
  springLoadedRelief.configZ,
  threeWay.configZ,
  threeWayBall.configZ,
  valve.configZ,
]);
