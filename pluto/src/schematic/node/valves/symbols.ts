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
  defineDummyToggle,
  defineStatic,
  defineToggle,
} from "@/schematic/node/common/define";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Primitive as BasePrimitive } from "@/schematic/node/common/primitive";
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

const angled = defineToggle({
  variant: "angledValve",
  name: "Angled",
  label: "Angled Valve",
  Primitive: Angled,
});
const angledRelief = defineDummyToggle({
  variant: "angledReliefValve",
  name: "Angled Relief",
  label: "Angled Relief Valve",
  Primitive: AngledRelief,
});
const angledSpringLoadedRelief = defineDummyToggle({
  variant: "angledSpringLoadedReliefValve",
  name: "Angled Spring Loaded Relief",
  label: "Angled Spring Loaded Relief Valve",
  Primitive: AngledSpringLoadedRelief,
});
const ball = defineToggle({
  variant: "ballValve",
  name: "Ball",
  label: "Ball Valve",
  Primitive: Ball,
});
const breather = defineDummyToggle({
  variant: "breatherValve",
  name: "Breather",
  label: "Breather Valve",
  Primitive: Breather,
});
const butterflyOne = defineToggle({
  variant: "butterflyValveOne",
  name: "Butterfly (Remote)",
  label: "Butterfly Valve (Remote)",
  Primitive: ButterflyOne,
});
const butterflyTwo = defineToggle({
  variant: "butterflyValveTwo",
  name: "Butterfly (Manual)",
  label: "Butterfly Valve (Manual)",
  Primitive: ButterflyTwo,
});
const check = defineStatic({
  variant: "checkValve",
  name: "Check",
  label: "Check Valve",
  Primitive: Check,
});
const checkWithArrow = defineStatic({
  variant: "checkValveWithArrow",
  name: "Check (Arrow)",
  label: "Check Valve",
  Primitive: CheckWithArrow,
});
const electricRegulator = defineStatic({
  variant: "electricRegulator",
  name: "Electric",
  label: "Electric Regulator",
  Primitive: ElectricRegulator,
});
const electricRegulatorMotorized = defineStatic({
  variant: "electricRegulatorMotorized",
  name: "Motorized",
  label: "Electric Regulator Motorized",
  Primitive: ElectricRegulatorMotorized,
});
const fourWay = defineToggle({
  variant: "fourWayValve",
  name: "Four Way",
  label: "Four Way Valve",
  Primitive: FourWay,
});
const gate = defineToggle({
  variant: "gateValve",
  name: "Gate",
  label: "Gate Valve",
  Primitive: Gate,
});
const isoCheck = defineStatic({
  variant: "isoCheckValve",
  name: "ISO Check",
  label: "ISO Check Valve",
  Primitive: IsoCheck,
});
const manual = defineDummyToggle({
  variant: "manualValve",
  name: "Manual",
  label: "Manual Valve",
  Primitive: Manual,
});
const needle = defineDummyToggle({
  variant: "needleValve",
  name: "Needle",
  label: "Needle Valve",
  Primitive: Needle,
});
const regulator = defineStatic({
  variant: "regulator",
  name: "Regulator",
  label: "Regulator",
  Primitive: Regulator,
});
const regulatorManual = defineStatic({
  variant: "regulatorManual",
  name: "Manual",
  label: "Manual Regulator",
  Primitive: RegulatorManual,
});
const relief = defineDummyToggle({
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
  defaultConfig: (t: Theming.Theme): SolenoidConfig => ({
    variant: "solenoidValve",
    color: t.colors.gray.l11,
    label: Label.defaultConfig("Solenoid Valve"),
    normallyOpen: false,
    ...BasePrimitive.ZERO_PROPS,
    ...Toggle.ZERO_TOGGLE_DEFAULTS,
  }),
  zIndex: 4,
};
const springLoadedRelief = defineDummyToggle({
  variant: "springLoadedReliefValve",
  name: "Spring Loaded Relief",
  label: "Spring Loaded Relief Valve",
  Primitive: SpringLoadedRelief,
});
const threeWay = defineToggle({
  variant: "threeWayValve",
  name: "Three Way",
  label: "Three Way",
  Primitive: ThreeWay,
});
const threeWayBall = defineToggle({
  variant: "threeWayBallValve",
  name: "Three-Way Ball",
  label: "Three-Way Ball Valve",
  Primitive: ThreeWayBall,
});
const valve = defineToggle({
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
