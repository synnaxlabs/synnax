// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";
import { z } from "zod";

import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Theming } from "@/theming";
import { removeProps } from "@/util/removeProps";
import {
  BoxForm,
  ButtonForm,
  CommonDummyToggleForm,
  CommonStyleForm,
  CommonToggleForm,
  CylinderForm,
  LightForm,
  OffPageReferenceForm,
  SetpointForm,
  SwitchForm,
  type SymbolFormProps,
  TankForm,
  TextBoxForm,
  ValueForm,
} from "@/vis/schematic/Forms";
import { Primitives } from "@/vis/schematic/primitives";
import {
  type CylinderProps,
  DEFAULT_BORDER_RADIUS,
  TextBox,
  type TextBoxProps,
} from "@/vis/schematic/primitives/Primitives";
import {
  Agitator,
  type AgitatorProps,
  AngledReliefValve,
  AngledSpringLoadedReliefValve,
  type AngledSpringLoadedReliefValveProps,
  AngledValve,
  type AngledValveProps,
  BallValve,
  type BallValveProps,
  Box,
  BoxPreview,
  type BoxProps,
  BreatherValve,
  type BreatherValveProps,
  BurstDisc,
  type BurstDiscProps,
  ButterflyValveOne,
  type ButterflyValveOneProps,
  ButterflyValveTwo,
  type ButterflyValveTwoProps,
  Button,
  ButtonPreview,
  type ButtonProps,
  Cap,
  type CapProps,
  CavityPump,
  type CavityPumpProps,
  CentrifugalCompressor,
  type CentrifugalCompressorProps,
  CheckValve,
  type CheckValveProps,
  Compressor,
  type CompressorProps,
  CrossBeamAgitator,
  type CrossBeamAgitatorProps,
  CrossJunction,
  type CrossJunctionProps,
  Cylinder,
  CylinderPreview,
  DiaphragmPump,
  type DiaphragmPumpProps,
  EjectionPump,
  type EjectionPumpProps,
  EjectorCompressor,
  type EjectorCompressorProps,
  ElectricRegulator,
  type ElectricRegulatorProps,
  Filter,
  type FilterProps,
  FlameArrestor,
  FlameArrestorDetonation,
  type FlameArrestorDetonationProps,
  FlameArrestorExplosion,
  type FlameArrestorExplosionProps,
  FlameArrestorFireRes,
  FlameArrestorFireResDetonation,
  type FlameArrestorFireResDetonationProps,
  type FlameArrestorFireResProps,
  type FlameArrestorProps,
  FlatBladeAgitator,
  type FlatBladeAgitatorProps,
  FlowmeterCoriolis,
  type FlowmeterCoriolisProps,
  FlowmeterElectromagnetic,
  type FlowmeterElectromagneticProps,
  FlowmeterFloatSensor,
  type FlowmeterFloatSensorProps,
  FlowmeterGeneral,
  type FlowmeterGeneralProps,
  FlowmeterNozzle,
  type FlowmeterNozzleProps,
  FlowmeterPositiveDisplacement,
  type FlowmeterPositiveDisplacementProps,
  FlowmeterPulse,
  type FlowmeterPulseProps,
  FlowmeterRingPiston,
  type FlowmeterRingPistonProps,
  FlowmeterTurbine,
  type FlowmeterTurbineProps,
  FlowmeterVariableArea,
  type FlowmeterVariableAreaProps,
  FlowmeterVenturi,
  type FlowmeterVenturiProps,
  FourWayValve,
  type FourWayValveProps,
  GateValve,
  type GateValveProps,
  HeatExchangerGeneral,
  type HeatExchangerGeneralProps,
  HeatExchangerM,
  type HeatExchangerMProps,
  HeatExchangerStraightTube,
  type HeatExchangerStraightTubeProps,
  HelicalAgitator,
  type HelicalAgitatorProps,
  ISOBurstDisc,
  type ISOBurstDiscProps,
  ISOCap,
  type ISOCapProps,
  ISOCheckValve,
  type ISOCheckValveProps,
  ISOFilter,
  type ISOFilterProps,
  type LabelExtensionProps,
  Light,
  type LightProps,
  LiquidRingCompressor,
  type LiquidRingCompressorProps,
  ManualValve,
  type ManualValveProps,
  NeedleValve,
  type NeedleValveProps,
  OffPageReference,
  OffPageReferencePreview,
  type OffPageReferenceProps,
  Orifice,
  OrificePlate,
  type OrificePlateProps,
  type OrificeProps,
  PaddleAgitator,
  type PaddleAgitatorProps,
  PistonPump,
  type PistonPumpProps,
  PropellerAgitator,
  type PropellerAgitatorProps,
  Pump,
  type PumpProps,
  Regulator,
  type RegulatorProps,
  ReliefValve,
  type ReliefValveProps,
  RollerVaneCompressor,
  type RollerVaneCompressorProps,
  RotaryMixer,
  type RotaryMixerProps,
  ScrewPump,
  type ScrewPumpProps,
  Setpoint,
  SetpointPreview,
  type SetpointProps,
  SolenoidValve,
  type SolenoidValveProps,
  SpringLoadedReliefValve,
  type SpringLoadedReliefValveProps,
  StaticMixer,
  type StaticMixerProps,
  Strainer,
  StrainerCone,
  type StrainerConeProps,
  type StrainerProps,
  Switch,
  type SwitchProps,
  type SymbolProps,
  Tank,
  TankPreview,
  type TankProps,
  TextBoxPreview,
  ThreeWayBallValve,
  type ThreeWayBallValveProps,
  ThreeWayValve,
  type ThreeWayValveProps,
  Thruster,
  type ThrusterProps,
  TJunction,
  type TJunctionProps,
  TurboCompressor,
  type TurboCompressorProps,
  VacuumPump,
  type VacuumPumpProps,
  Value,
  ValuePreview,
  type ValueProps,
  Valve,
  type ValveProps,
  Vent,
  type VentProps,
} from "@/vis/schematic/Symbols";

export interface Spec<P extends object> {
  key: Variant;
  name: string;
  Form: FC<SymbolFormProps>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<SymbolProps<P>>;
  zIndex: number;
}

const Z_INDEX_UPPER = 4;
const Z_INDEX_LOWER = 2;

const VARIANTS = [
  "agitator",
  "angledReliefValve",
  "angledSpringLoadedReliefValve",
  "angledValve",
  "ballValve",
  "threeWayBallValve",
  "gateValve",
  "butterflyValveOne",
  "butterflyValveTwo",
  "breatherValve",
  "offPageReference",
  "box",
  "burstDisc",
  "isoBurstDisc",
  "button",
  "cap",
  "cavityPump",
  "checkValve",
  "cylinder",
  "crossBeamAgitator",
  "electricRegulator",
  "filter",
  "flatBladeAgitator",
  "flowmeterGeneral",
  "flowmeterElectromagnetic",
  "flowmeterVariableArea",
  "flowmeterCoriolis",
  "flowmeterNozzle",
  "flowmeterVenturi",
  "flowmeterRingPiston",
  "flowmeterPositiveDisplacement",
  "flowmeterTurbine",
  "flowmeterPulse",
  "flowmeterFloatSensor",
  "fourWayValve",
  "helicalAgitator",
  "isoCap",
  "isoCheckValve",
  "isoFilter",
  "light",
  "manualValve",
  "needleValve",
  "orifice",
  "orificePlate",
  "paddleAgitator",
  "propellerAgitator",
  "pistonPump",
  "pump",
  "regulator",
  "reliefValve",
  "rotaryMixer",
  "screwPump",
  "setpoint",
  "solenoidValve",
  "springLoadedReliefValve",
  "staticMixer",
  "switch",
  "tank",
  "textBox",
  "threeWayValve",
  "vacuumPump",
  "value",
  "valve",
  "vent",
  "tJunction",
  "crossJunction",
  "heatExchangerGeneral",
  "heatExchangerM",
  "heatExchangerStraightTube",
  "diaphragmPump",
  "ejectionPump",
  "compressor",
  "turboCompressor",
  "rollerVaneCompressor",
  "liquidRingCompressor",
  "ejectorCompressor",
  "centrifugalCompressor",
  "flameArrestor",
  "flameArrestorDetonation",
  "flameArrestorExplosion",
  "flameArrestorFireRes",
  "flameArrestorFireResDetonation",
  "thruster",
  "strainer",
  "strainerCone",
] as const;

export const variantZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof variantZ>;

const ZERO_PROPS = { orientation: "left" as const, scale: 1 };
const ZERO_NUMERIC_STRINGER_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("string", {
    connections: [
      { from: "valueStream", to: "rollingAverage" },
      { from: "rollingAverage", to: "stringifier" },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      stringifier: telem.stringifyNumber({ precision: 2, notation: "standard" }),
    },
    outlet: "stringifier",
  }),
};

const ZERO_NUMERIC_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("number", {
    connections: [],
    segments: { valueStream: telem.streamChannelValue({ channel: 0 }) },
    outlet: "valueStream",
  }),
};

const ZERO_NUMERIC_SINK_PROPS = {
  ...ZERO_PROPS,
  sink: telem.sinkPipeline("number", {
    connections: [],
    segments: { setter: control.setChannelValue({ channel: 0 }) },
    inlet: "setter",
  }),
};

const ZERO_BOOLEAN_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("boolean", {
    connections: [{ from: "valueStream", to: "threshold" }],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
    },
    outlet: "threshold",
  }),
};

const ZERO_BOOLEAN_SINK_PROPS = {
  ...ZERO_PROPS,
  control: { show: true },
  sink: telem.sinkPipeline("boolean", {
    connections: [{ from: "setpoint", to: "setter" }],
    segments: {
      setter: control.setChannelValue({ channel: 0 }),
      setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
    },
    inlet: "setpoint",
  }),
};

const ZERO_TOGGLE_PROPS = { ...ZERO_BOOLEAN_SOURCE_PROPS, ...ZERO_BOOLEAN_SINK_PROPS };

const ZERO_DUMMY_TOGGLE_PROPS = { ...ZERO_PROPS, enabled: false, clickable: false };

type zeroLabelReturn = { label: LabelExtensionProps };

const zeroLabel = (label: string): zeroLabelReturn => ({
  label: {
    label,
    level: "p",
    orientation: "top",
    maxInlineSize: 150,
    align: "center",
    direction: "x",
  },
});

const ZERO_DIMENSIONS = { width: 125, height: 200 };

const ZERO_BOX_PROPS = { dimensions: ZERO_DIMENSIONS };

const ZERO_BOX_BORDER_RADIUS = 3;

const threeWayValve: Spec<ThreeWayValveProps> = {
  name: "Three Way Valve",
  key: "threeWayValve",
  Form: CommonToggleForm,
  Symbol: ThreeWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Three Way Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ThreeWayValve,
  zIndex: Z_INDEX_UPPER,
};

const valve: Spec<ValveProps> = {
  name: "Valve",
  key: "valve",
  Form: CommonToggleForm,
  Symbol: Valve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Valve,
  zIndex: Z_INDEX_UPPER,
};

const solenoidValve: Spec<SolenoidValveProps> = {
  name: "Solenoid Valve",
  key: "solenoidValve",
  Form: CommonToggleForm,
  Symbol: SolenoidValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Solenoid Valve"),
    normallyOpen: false,
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.SolenoidValve,
  zIndex: Z_INDEX_UPPER,
};

const fourWayValve: Spec<FourWayValveProps> = {
  name: "Four Way Valve",
  key: "fourWayValve",
  Form: CommonToggleForm,
  Symbol: FourWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Four Way Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.FourWayValve,
  zIndex: Z_INDEX_UPPER,
};

const angledValve: Spec<AngledValveProps> = {
  name: "Angled Valve",
  key: "angledValve",
  Form: CommonToggleForm,
  Symbol: AngledValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.AngledValve,
  zIndex: Z_INDEX_UPPER,
};

const ballValve: Spec<BallValveProps> = {
  name: "Ball Valve",
  key: "ballValve",
  Form: CommonToggleForm,
  Symbol: BallValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Ball Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.BallValve,
  zIndex: Z_INDEX_UPPER,
};

const threeWayBallValve: Spec<ThreeWayBallValveProps> = {
  name: "Three-Way Ball Valve",
  key: "threeWayBallValve",
  Form: CommonToggleForm,
  Symbol: ThreeWayBallValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Three-Way Ball Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ThreeWayBallValve,
  zIndex: Z_INDEX_UPPER,
};

const gateValve: Spec<GateValveProps> = {
  name: "Gate Valve",
  key: "gateValve",
  Form: CommonToggleForm,
  Symbol: GateValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Gate Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.GateValve,
  zIndex: Z_INDEX_UPPER,
};

const butterflyValveOne: Spec<ButterflyValveOneProps> = {
  name: "Butterfly Valve (Remote)",
  key: "butterflyValveOne",
  Form: CommonToggleForm,
  Symbol: ButterflyValveOne,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Butterfly Valve (Remote)"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ButterflyValveOne,
  zIndex: Z_INDEX_UPPER,
};

const butterflyValveTwo: Spec<ButterflyValveTwoProps> = {
  name: "Butterfly Valve (Manual)",
  key: "butterflyValveTwo",
  Form: CommonDummyToggleForm,
  Symbol: ButterflyValveTwo,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Butterfly Valve (Manual)"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: ({ clickable, ...props }) => Primitives.ButterflyValveTwo(props),
  zIndex: Z_INDEX_UPPER,
};

const breatherValve: Spec<BreatherValveProps> = {
  name: "Breather Valve",
  key: "breatherValve",
  Form: CommonDummyToggleForm,
  Symbol: BreatherValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Breather Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.BreatherValve,
  zIndex: Z_INDEX_UPPER,
};

const pump: Spec<PumpProps> = {
  name: "Pump",
  key: "pump",
  Form: CommonToggleForm,
  Symbol: Pump,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Pump,
  zIndex: Z_INDEX_UPPER,
};

const screwPump: Spec<ScrewPumpProps> = {
  name: "Screw Pump",
  key: "screwPump",
  Form: CommonToggleForm,
  Symbol: ScrewPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Screw Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ScrewPump,
  zIndex: Z_INDEX_UPPER,
};

const tank: Spec<TankProps> = {
  name: "Tank",
  key: "tank",
  Form: TankForm,
  Symbol: Tank,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    backgroundColor: t.colors.gray.l1.setAlpha(0).rgba255,
    ...zeroLabel("Tank"),
    borderRadius: DEFAULT_BORDER_RADIUS,
    ...ZERO_BOX_PROPS,
    ...ZERO_PROPS,
  }),
  Preview: TankPreview,
  zIndex: Z_INDEX_LOWER,
};

const cylinder: Spec<CylinderProps> = {
  name: "Cylinder",
  key: "cylinder",
  Form: CylinderForm,
  Symbol: Cylinder,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    backgroundColor: t.colors.gray.l1.setAlpha(0).rgba255,
    ...zeroLabel("cylinder"),
    dimensions: {
      width: 66,
      height: 181,
    },
    ...ZERO_PROPS,
  }),
  Preview: CylinderPreview,
  zIndex: Z_INDEX_LOWER,
};

const box: Spec<BoxProps> = {
  name: "Box",
  key: "box",
  Form: BoxForm,
  Symbol: Box,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    backgroundColor: t.colors.gray.l1.setAlpha(0).rgba255,
    ...zeroLabel("Box"),
    borderRadius: ZERO_BOX_BORDER_RADIUS,
    ...ZERO_BOX_PROPS,
    ...ZERO_PROPS,
  }),
  Preview: BoxPreview,
  zIndex: Z_INDEX_LOWER,
};

const reliefValve: Spec<ReliefValveProps> = {
  name: "Relief Valve",
  key: "reliefValve",
  Form: CommonDummyToggleForm,
  Symbol: ReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.ReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const springLoadedReliefValve: Spec<SpringLoadedReliefValveProps> = {
  name: "Spring Loaded Relief Valve",
  key: "springLoadedReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: SpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Spring Loaded Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.SpringLoadedReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const angledSpringLoadedReliefValve: Spec<AngledSpringLoadedReliefValveProps> = {
  name: "Angled Spring Loaded Relief Valve",
  key: "angledSpringLoadedReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: AngledSpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Spring Loaded Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.AngledSpringLoadedReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const regulator: Spec<RegulatorProps> = {
  name: "Regulator",
  key: "regulator",
  Form: CommonStyleForm,
  Symbol: Regulator,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Regulator,
  zIndex: Z_INDEX_UPPER,
};

const electricRegulator: Spec<ElectricRegulatorProps> = {
  name: "Electric Regulator",
  key: "electricRegulator",
  Form: CommonStyleForm,
  Symbol: ElectricRegulator,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Electric Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ElectricRegulator,
  zIndex: Z_INDEX_UPPER,
};

const burstDisc: Spec<BurstDiscProps> = {
  name: "Burst Disc",
  key: "burstDisc",
  Form: CommonStyleForm,
  Symbol: BurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.BurstDisc,
  zIndex: Z_INDEX_UPPER,
};

const isoBurstDisc: Spec<ISOBurstDiscProps> = {
  name: "ISO Burst Disc",
  key: "isoBurstDisc",
  Form: CommonStyleForm,
  Symbol: ISOBurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOBurstDisc,
  zIndex: Z_INDEX_UPPER,
};

const cap: Spec<CapProps> = {
  name: "Cap",
  key: "cap",
  Form: CommonStyleForm,
  Symbol: Cap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cap"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Cap,
  zIndex: Z_INDEX_UPPER,
};

const isoCap: Spec<ISOCapProps> = {
  name: "ISO Cap",
  key: "isoCap",
  Form: CommonStyleForm,
  Symbol: ISOCap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Cap"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOCap,
  zIndex: Z_INDEX_UPPER,
};

const manualValve: Spec<ManualValveProps> = {
  name: "Manual Valve",
  key: "manualValve",
  Form: CommonDummyToggleForm,
  Symbol: ManualValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Manual Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.ManualValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const orificePlate: Spec<OrificePlateProps> = {
  name: "Orifice Plate",
  key: "orificePlate",
  Form: CommonStyleForm,
  Symbol: OrificePlate,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Orifice Plate"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.OrificePlate,
  zIndex: Z_INDEX_UPPER,
};

const isoFilter: Spec<ISOFilterProps> = {
  name: "ISO Filter",
  key: "isoFilter",
  Form: CommonStyleForm,
  Symbol: ISOFilter,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Filter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOFilter,
  zIndex: Z_INDEX_UPPER,
};

const filter: Spec<FilterProps> = {
  name: "Filter",
  key: "filter",
  Form: CommonStyleForm,
  Symbol: Filter,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Filter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Filter,
  zIndex: Z_INDEX_UPPER,
};

const needleValve: Spec<NeedleValveProps> = {
  name: "Needle Valve",
  key: "needleValve",
  Form: CommonDummyToggleForm,
  Symbol: NeedleValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Needle Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.NeedleValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const checkValve: Spec<CheckValveProps> = {
  name: "Check Valve",
  key: "checkValve",
  Form: CommonStyleForm,
  Symbol: CheckValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Check Valve"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.CheckValve,
  zIndex: Z_INDEX_UPPER,
};

const orifice: Spec<OrificeProps> = {
  name: "Orifice",
  key: "orifice",
  Form: CommonStyleForm,
  Symbol: Orifice,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Orifice"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Orifice,
  zIndex: Z_INDEX_UPPER,
};

const angledReliefValve: Spec<ReliefValveProps> = {
  name: "Angled Relief Valve",
  key: "angledReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: AngledReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.AngledReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const value: Spec<ValueProps> = {
  name: "Value",
  key: "value",
  Form: ValueForm,
  Symbol: Value,
  Preview: ValuePreview,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    units: "psi",
    level: "h5",
    inlineSize: 70,
    ...zeroLabel("Value"),
    ...ZERO_PROPS,
    telem: ZERO_NUMERIC_STRINGER_SOURCE_PROPS.source,
  }),
  zIndex: Z_INDEX_UPPER,
};

const button: Spec<ButtonProps> = {
  name: "Button",
  key: "button",
  Symbol: Button,
  Form: ButtonForm,
  Preview: ButtonPreview,
  defaultProps: (t) => ({
    color: t.colors.primary.z.rgba255,
    ...zeroLabel("Button"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    onClickDelay: 0,
    scale: null,
  }),
  zIndex: Z_INDEX_UPPER,
};

const switch_: Spec<SwitchProps> = {
  name: "Switch",
  key: "switch",
  Symbol: Switch,
  Form: SwitchForm,
  defaultProps: () => ({
    ...zeroLabel("Switch"),
    ...ZERO_TOGGLE_PROPS,
    scale: null,
  }),
  Preview: Primitives.Switch,
  zIndex: Z_INDEX_UPPER,
};

const vacuumPump: Spec<VacuumPumpProps> = {
  name: "Vacuum Pump",
  key: "vacuumPump",
  Symbol: VacuumPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Vacuum Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.VacuumPump,
  zIndex: Z_INDEX_UPPER,
};

const compressor: Spec<CompressorProps> = {
  name: "Compressor",
  key: "compressor",
  Symbol: Compressor,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Compressor,
  zIndex: Z_INDEX_UPPER,
};

const cavityPump: Spec<CavityPumpProps> = {
  name: "Cavity Pump",
  key: "cavityPump",
  Symbol: CavityPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cavity Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.CavityPump,
  zIndex: Z_INDEX_UPPER,
};

const pistonPump: Spec<PistonPumpProps> = {
  name: "Piston Pump",
  key: "pistonPump",
  Symbol: PistonPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Piston Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.PistonPump,
  zIndex: Z_INDEX_UPPER,
};

const staticMixer: Spec<StaticMixerProps> = {
  name: "Static Mixer",
  key: "staticMixer",
  Symbol: StaticMixer,
  Form: CommonStyleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Static Mixer"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.StaticMixer,
  zIndex: Z_INDEX_UPPER,
};

const rotaryMixer: Spec<RotaryMixerProps> = {
  name: "Rotary Mixer",
  key: "rotaryMixer",
  Symbol: RotaryMixer,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Rotary Mixer"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.RotaryMixer,
  zIndex: Z_INDEX_UPPER,
};

const light: Spec<LightProps> = {
  name: "Light",
  key: "light",
  Symbol: Light,
  Form: LightForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Light"),
    ...ZERO_BOOLEAN_SOURCE_PROPS,
  }),
  Preview: Primitives.Light,
  zIndex: Z_INDEX_UPPER,
};

const setpoint: Spec<SetpointProps> = {
  name: "Setpoint",
  key: "setpoint",
  Symbol: Setpoint,
  Form: SetpointForm,
  defaultProps: (t) => ({
    units: "mV",
    color: t.colors.gray.l9.rgba255,
    size: "small",
    ...zeroLabel("Setpoint"),
    ...ZERO_NUMERIC_SOURCE_PROPS,
    ...ZERO_NUMERIC_SINK_PROPS,
  }),
  Preview: SetpointPreview,
  zIndex: Z_INDEX_UPPER,
};

const agitator: Spec<AgitatorProps> = {
  name: "Agitator",
  key: "agitator",
  Symbol: Agitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Agitator,
  zIndex: Z_INDEX_UPPER,
};

const propellerAgitator: Spec<PropellerAgitatorProps> = {
  name: "Propeller Agitator",
  key: "propellerAgitator",
  Symbol: PropellerAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Propeller Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.PropellerAgitator,
  zIndex: Z_INDEX_UPPER,
};

const flatBladeAgitator: Spec<FlatBladeAgitatorProps> = {
  name: "Flat Blade Agitator",
  key: "flatBladeAgitator",
  Symbol: FlatBladeAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flat Blade Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.FlatBladeAgitator,
  zIndex: Z_INDEX_UPPER,
};

const paddleAgitator: Spec<PaddleAgitatorProps> = {
  name: "Paddle Agitator",
  key: "paddleAgitator",
  Symbol: PaddleAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Paddle Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.PaddleAgitator,
  zIndex: Z_INDEX_UPPER,
};

const crossBeamAgitator: Spec<CrossBeamAgitatorProps> = {
  name: "Cross Beam Agitator",
  key: "crossBeamAgitator",
  Symbol: CrossBeamAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cross Beam Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.CrossBeamAgitator,
  zIndex: Z_INDEX_UPPER,
};

const helicalAgitator: Spec<HelicalAgitatorProps> = {
  name: "Helical Agitator",
  key: "helicalAgitator",
  Symbol: HelicalAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Helical Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.HelicalAgitator,
  zIndex: Z_INDEX_UPPER,
};

const textBox: Spec<TextBoxProps> = {
  name: "Text Box",
  key: "textBox",
  Symbol: TextBox,
  Form: TextBoxForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    autoFit: true,
    align: "center",
    ...zeroLabel("Text Box"),
    ...ZERO_PROPS,
    ...ZERO_BOX_PROPS,
    level: "p",
    text: "Text Box",
    width: 75,
  }),
  Preview: TextBoxPreview,
  zIndex: Z_INDEX_UPPER,
};

const offPageReference: Spec<OffPageReferenceProps> = {
  name: "Off Page Reference",
  key: "offPageReference",
  Form: OffPageReferenceForm,
  Symbol: OffPageReference,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    orientation: "right",
    ...zeroLabel("Off Page Reference"),
  }),
  Preview: OffPageReferencePreview,
  zIndex: Z_INDEX_UPPER,
};

const isoCheckValve: Spec<ISOCheckValveProps> = {
  name: "ISO Check Valve",
  key: "isoCheckValve",
  Form: CommonStyleForm,
  Symbol: ISOCheckValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Check Valve"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOCheckValve,
  zIndex: Z_INDEX_UPPER,
};

const vent: Spec<VentProps> = {
  name: "Vent",
  key: "vent",
  Form: CommonStyleForm,
  Symbol: Vent,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Vent"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Vent,
  zIndex: Z_INDEX_UPPER,
};

const tJunction: Spec<TJunctionProps> = {
  name: "T Junction",
  key: "tJunction",
  Form: CommonStyleForm,
  Symbol: TJunction,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel(""),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.TJunction,
  zIndex: Z_INDEX_UPPER + 20,
};

const crossJunction: Spec<CrossJunctionProps> = {
  name: "Cross Junction",
  key: "crossJunction",
  Form: CommonStyleForm,
  Symbol: CrossJunction,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel(""),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.CrossJunction,
  zIndex: Z_INDEX_UPPER + 20,
};

const flowmeterGeneral: Spec<FlowmeterGeneralProps> = {
  name: "Flowmeter General",
  key: "flowmeterGeneral",
  Form: CommonStyleForm,
  Symbol: FlowmeterGeneral,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("General Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterGeneral,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterElectromagnetic: Spec<FlowmeterElectromagneticProps> = {
  name: "Flowmeter Electromagnetic",
  key: "flowmeterElectromagnetic",
  Form: CommonStyleForm,
  Symbol: FlowmeterElectromagnetic,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Electromagnetic Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterElectromagnetic,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterVariableArea: Spec<FlowmeterVariableAreaProps> = {
  name: "Flowmeter Variable Area",
  key: "flowmeterVariableArea",
  Form: CommonStyleForm,
  Symbol: FlowmeterVariableArea,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Variable Area Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterVariableArea,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterCoriolis: Spec<FlowmeterCoriolisProps> = {
  name: "Flowmeter Coriolis",
  key: "flowmeterCoriolis",
  Form: CommonStyleForm,
  Symbol: FlowmeterCoriolis,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Coriolis Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterCoriolis,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterNozzle: Spec<FlowmeterNozzleProps> = {
  name: "Flowmeter Nozzle",
  key: "flowmeterNozzle",
  Form: CommonStyleForm,
  Symbol: FlowmeterNozzle,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Nozzle Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterNozzle,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterVenturi: Spec<FlowmeterVenturiProps> = {
  name: "Flowmeter Venturi",
  key: "flowmeterVenturi",
  Form: CommonStyleForm,
  Symbol: FlowmeterVenturi,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Venturi Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterVenturi,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterRingPiston: Spec<FlowmeterRingPistonProps> = {
  name: "Flowmeter Ring Piston",
  key: "flowmeterRingPiston",
  Form: CommonStyleForm,
  Symbol: FlowmeterRingPiston,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Ring Piston Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterRingPiston,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterPositiveDisplacement: Spec<FlowmeterPositiveDisplacementProps> = {
  name: "Flowmeter Positive Displacement",
  key: "flowmeterPositiveDisplacement",
  Form: CommonStyleForm,
  Symbol: FlowmeterPositiveDisplacement,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Positive Displacement Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterPositiveDisplacement,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterTurbine: Spec<FlowmeterTurbineProps> = {
  name: "Flowmeter Turbine",
  key: "flowmeterTurbine",
  Form: CommonStyleForm,
  Symbol: FlowmeterTurbine,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Turbine Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterTurbine,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterPulse: Spec<FlowmeterPulseProps> = {
  name: "Flowmeter Pulse",
  key: "flowmeterPulse",
  Form: CommonStyleForm,
  Symbol: FlowmeterPulse,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Pulse Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterPulse,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterFloatSensor: Spec<FlowmeterFloatSensorProps> = {
  name: "Flowmeter Float Sensor",
  key: "flowmeterFloatSensor",
  Form: CommonStyleForm,
  Symbol: FlowmeterFloatSensor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Float Sensor Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterFloatSensor,
  zIndex: Z_INDEX_UPPER,
};

const heatExchangerGeneral: Spec<HeatExchangerGeneralProps> = {
  name: "Heat Exchanger General",
  key: "heatExchangerGeneral",
  Form: CommonStyleForm,
  Symbol: HeatExchangerGeneral,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("General Heat Exchanger"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeatExchangerGeneral,
  zIndex: Z_INDEX_UPPER,
};

const heatExchangerM: Spec<HeatExchangerMProps> = {
  name: "Heat Exchanger M",
  key: "heatExchangerM",
  Form: CommonStyleForm,
  Symbol: HeatExchangerM,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("M Heat Exchanger"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeatExchangerM,
  zIndex: Z_INDEX_UPPER,
};

const heatExchangerStraightTube: Spec<HeatExchangerStraightTubeProps> = {
  name: "Heat Exchanger Straight Tube",
  key: "heatExchangerStraightTube",
  Form: CommonStyleForm,
  Symbol: HeatExchangerStraightTube,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Straight Tube Heat Exchanger"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeatExchangerStraightTube,
  zIndex: Z_INDEX_UPPER,
};

const turboCompressor: Spec<TurboCompressorProps> = {
  name: "Turbo Compressor",
  key: "turboCompressor",
  Form: CommonToggleForm,
  Symbol: TurboCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Turbo Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.TurboCompressor,
  zIndex: Z_INDEX_UPPER,
};

const rollerVaneCompressor: Spec<RollerVaneCompressorProps> = {
  name: "Roller Vane Compressor",
  key: "rollerVaneCompressor",
  Form: CommonToggleForm,
  Symbol: RollerVaneCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Roller Vane Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.RollerVaneCompressor,
  zIndex: Z_INDEX_UPPER,
};

const liquidRingCompressor: Spec<LiquidRingCompressorProps> = {
  name: "Liquid Ring Compressor",
  key: "liquidRingCompressor",
  Form: CommonToggleForm,
  Symbol: LiquidRingCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Liquid Ring Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.LiquidRingCompressor,
  zIndex: Z_INDEX_UPPER,
};

const ejectorCompressor: Spec<EjectorCompressorProps> = {
  name: "Ejector Compressor",
  key: "ejectorCompressor",
  Form: CommonToggleForm,
  Symbol: EjectorCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Ejector Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.EjectorCompressor,
  zIndex: Z_INDEX_UPPER,
};

const centrifugalCompressor: Spec<CentrifugalCompressorProps> = {
  name: "Centrifugal Compressor",
  key: "centrifugalCompressor",
  Form: CommonToggleForm,
  Symbol: CentrifugalCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Centrifugal Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.CentrifugalCompressor,
  zIndex: Z_INDEX_UPPER,
};

const diaphragmPump: Spec<DiaphragmPumpProps> = {
  name: "Diaphragm Pump",
  key: "diaphragmPump",
  Form: CommonToggleForm,
  Symbol: DiaphragmPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Diaphragm Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.DiaphragmPump,
  zIndex: Z_INDEX_UPPER,
};

const ejectionPump: Spec<EjectionPumpProps> = {
  name: "Ejection Pump",
  key: "ejectionPump",
  Form: CommonToggleForm,
  Symbol: EjectionPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Ejection Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.EjectionPump,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestor: Spec<FlameArrestorProps> = {
  name: "Flame Arrestor",
  key: "flameArrestor",
  Form: CommonStyleForm,
  Symbol: FlameArrestor,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flame Arrestor"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestor,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorExplosion: Spec<FlameArrestorExplosionProps> = {
  name: "Flame Arrestor (Explosion-Proof)",
  key: "flameArrestorExplosion",
  Form: CommonStyleForm,
  Symbol: FlameArrestorExplosion,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flame Arrestor (Explosion-Proof)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorExplosion,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorDetonation: Spec<FlameArrestorDetonationProps> = {
  name: "Flame Arrestor (Detonation-Proof)",
  key: "flameArrestorDetonation",
  Form: CommonStyleForm,
  Symbol: FlameArrestorDetonation,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flame Arrestor (Detonation-Proof)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorDetonation,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorFireRes: Spec<FlameArrestorFireResProps> = {
  name: "Flame Arrestor (Fire Resistant)",
  key: "flameArrestorFireRes",
  Form: CommonStyleForm,
  Symbol: FlameArrestorFireRes,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flame Arrestor (Fire Resistant)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorFireRes,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorFireResDetonation: Spec<FlameArrestorFireResDetonationProps> = {
  name: "Flame Arrestor (Fire Resistant and Detonation-Proof)",
  key: "flameArrestorFireResDetonation",
  Form: CommonStyleForm,
  Symbol: FlameArrestorFireResDetonation,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flame Arrestor (Fire Resistant and Detonation-Proof)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorFireResDetonation,
  zIndex: Z_INDEX_UPPER,
};

const thruster: Spec<ThrusterProps> = {
  name: "Thruster",
  key: "thruster",
  Form: CommonToggleForm,
  Symbol: Thruster,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Thruster"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Thruster,
  zIndex: Z_INDEX_UPPER,
};

const strainer: Spec<StrainerProps> = {
  name: "Strainer",
  key: "strainer",
  Form: CommonStyleForm,
  Symbol: Strainer,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Strainer"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Strainer,
  zIndex: Z_INDEX_UPPER,
};

const strainerCone: Spec<StrainerConeProps> = {
  name: "Strainer Cone",
  key: "strainerCone",
  Form: CommonStyleForm,
  Symbol: StrainerCone,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Strainer Cone"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.StrainerCone,
  zIndex: Z_INDEX_UPPER,
};

export const SYMBOLS: Record<Variant, Spec<any>> = {
  value,
  button,
  tank,
  tJunction,
  crossJunction,
  switch: switch_,
  offPageReference,
  light,
  setpoint,
  box,
  textBox,
  valve,
  solenoidValve,
  threeWayValve,
  fourWayValve,
  angledValve,
  ballValve,
  threeWayBallValve,
  gateValve,
  butterflyValveOne,
  butterflyValveTwo,
  breatherValve,
  manualValve,
  needleValve,
  reliefValve,
  angledReliefValve,
  checkValve,
  regulator,
  electricRegulator,
  springLoadedReliefValve,
  angledSpringLoadedReliefValve,
  pump,
  pistonPump,
  screwPump,
  cavityPump,
  diaphragmPump,
  ejectionPump,
  vacuumPump,
  compressor,
  turboCompressor,
  rollerVaneCompressor,
  liquidRingCompressor,
  ejectorCompressor,
  centrifugalCompressor,
  staticMixer,
  rotaryMixer,
  burstDisc,
  isoBurstDisc,
  cap,
  isoCap,
  filter,
  isoFilter,
  orifice,
  orificePlate,
  agitator,
  propellerAgitator,
  flatBladeAgitator,
  paddleAgitator,
  crossBeamAgitator,
  helicalAgitator,
  isoCheckValve,
  vent,
  cylinder,
  flowmeterGeneral,
  flowmeterElectromagnetic,
  flowmeterVariableArea,
  flowmeterCoriolis,
  flowmeterNozzle,
  flowmeterVenturi,
  flowmeterRingPiston,
  flowmeterPositiveDisplacement,
  flowmeterTurbine,
  flowmeterPulse,
  flowmeterFloatSensor,
  heatExchangerGeneral,
  heatExchangerM,
  heatExchangerStraightTube,
  flameArrestor,
  flameArrestorExplosion,
  flameArrestorDetonation,
  flameArrestorFireRes,
  flameArrestorFireResDetonation,
  thruster,
  strainer,
  strainerCone,
};
